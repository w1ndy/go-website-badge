package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// MonitorWebsite sets a website to monitor
type MonitorWebsite struct {
	URL                string `json:"URL"`
	Identifier         string `json:"Identifier"`
	InsecureSkipVerify bool   `json:"InsecureSkipVerify"`
	Timeout            int64  `json:"Timeout"`
	Interval           int64  `json:"Interval"`
	Proxy              string `json:"Proxy"`
	Mode               string `json:"Mode"`

	Result             bool        `json:"-"`
	ResultCount        int64       `json:"-"`
	ResultSuccessCount int64       `json:"-"`
	LastSeen           time.Time   `json:"-"`
	Logger             *log.Entry  `json:"-"`
	Lock               *sync.Mutex `json:"-"`
	Channel            chan bool   `json:"-"`
}

// Configuration describes monitor settings including websites, interval (seconds), and timeout (seconds)
type Configuration struct {
	Websites []MonitorWebsite `json:"Websites"`
	Interval int64            `json:"Interval"`
	Timeout  int64            `json:"Timeout"`
}

var config = &Configuration{
	Interval: 30,
	Timeout:  5,
}

func init() {
	confPath := flag.String("conf", "config.json", "path to configuration")
	logLevel := flag.Uint("loglevel", 4, "log level 0-6")
	flag.Parse()

	log.SetLevel(log.Level(*logLevel))

	log.Infof("loading configurations from %s...", *confPath)
	confFile, err := ioutil.ReadFile(*confPath)
	if err != nil {
		log.Panic(err)
	}

	err = json.Unmarshal(confFile, &config)
	if err != nil {
		log.Panic(err)
	}

	for i := range config.Websites {
		site := &config.Websites[i]
		site.Result = false
		site.Lock = &sync.Mutex{}
		site.Logger = log.WithFields(log.Fields{
			"Mode":       site.Mode,
			"URL":        site.URL,
			"Identifier": site.Identifier,
		})
		site.Logger.Info("monitor initialized")
	}
}

func parseProxyURL(proxyURL string, logger *log.Entry) *url.URL {
	url, err := url.Parse(proxyURL)
	if err != nil {
		logger.WithFields(log.Fields{
			"proxy": proxyURL,
			"err":   err,
		}).Panic("unable to parse proxy address as URL!")
	}
	return url
}

func testTCP(site *MonitorWebsite, interval, timeout time.Duration) {
	var d proxy.Dialer
	d = &net.Dialer{
		Timeout: timeout,
	}

	// TODO: this does not work as expected because with proxy.Dialer the dials always succeed
	if site.Proxy != "" {
		var err error
		url := parseProxyURL(site.Proxy, site.Logger)
		d, err = proxy.FromURL(url, d)
		if err != nil {
			site.Logger.WithFields(log.Fields{
				"proxy": site.Proxy,
				"err":   err,
			}).Panic("unable to parse proxy address!")
		}
	}

	for {
		conn, err := d.Dial("tcp", site.URL)

		site.Lock.Lock()

		if err == nil {
			site.Logger.Trace("site is up")
			if !site.Result {
				site.Logger.Info("site restored")
			}

			site.Result = true
			site.ResultSuccessCount++
			site.LastSeen = time.Now()
		} else {
			loggerWithError := site.Logger.WithFields(log.Fields{
				"err": err,
			})

			loggerWithError.Trace("site is down")
			if site.Result {
				loggerWithError.Info("site went down!")
			}

			site.Result = false
		}

		site.ResultCount++
		site.Lock.Unlock()
		conn.Close()
		time.Sleep(interval)
	}
}

func testHTTP(site *MonitorWebsite, interval, timeout time.Duration) {
	var proxy func(*http.Request) (*url.URL, error)

	if site.Proxy != "" {
		url := parseProxyURL(site.Proxy, site.Logger)
		proxy = http.ProxyURL(url)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: site.InsecureSkipVerify,
		},
		Proxy: proxy,
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	for {
		site.Logger.Trace("testing connectivity...")

		resp, err := client.Get(site.URL)

		site.Lock.Lock()

		if err == nil && resp.StatusCode == 200 {
			site.Logger.Trace("site is up")
			if !site.Result {
				site.Logger.Info("site restored.")
			}

			site.Result = true
			site.ResultSuccessCount++
			site.LastSeen = time.Now()
		} else {
			code := -1
			if resp != nil {
				code = resp.StatusCode
			}

			loggerWithError := site.Logger.WithFields(log.Fields{
				"err":  err,
				"code": code,
			})

			loggerWithError.Trace("site is down")
			if site.Result {
				loggerWithError.Info("site went down!")
			}

			site.Result = false
		}

		site.ResultCount++
		site.Lock.Unlock()
		time.Sleep(interval)
	}
}

func testPassive(site *MonitorWebsite, timeout time.Duration) {
	for {
		select {
		case <-time.After(timeout):
			site.Lock.Lock()
			site.Logger.Trace("site is down")
			if site.Result {
				site.Logger.Info("site went down!")
			}
			site.Result = false
		case <-site.Channel:
			site.Lock.Lock()
			site.Logger.Trace("site is up")
			if !site.Result {
				site.Logger.Info("site restored")
			}
			site.Result = true
			site.ResultSuccessCount++
			site.LastSeen = time.Now()
		}
		site.ResultCount++
		site.Lock.Unlock()
	}
}

func main() {
	for k := range config.Websites {
		var interval, timeout time.Duration

		if config.Websites[k].Interval != 0 {
			interval = time.Duration(time.Duration(config.Websites[k].Interval) * time.Second)
		} else {
			interval = time.Duration(time.Duration(config.Interval) * time.Second)
		}
		if config.Websites[k].Timeout != 0 {
			timeout = time.Duration(time.Duration(config.Websites[k].Timeout) * time.Second)
		} else {
			timeout = time.Duration(time.Duration(config.Timeout) * time.Second)
		}

		switch config.Websites[k].Mode {
		case "HTTP":
			go testHTTP(&config.Websites[k], interval, timeout)
		case "TCP":
			go testTCP(&config.Websites[k], interval, timeout)
		case "Passive":
			config.Websites[k].Channel = make(chan bool)
			go testPassive(&config.Websites[k], timeout)
		default:
			go testHTTP(&config.Websites[k], interval, timeout)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/", func(context *gin.Context) {
		context.JSON(200, map[string]string{
			"running": "yes",
		})
	})

	for k := range config.Websites {
		site := &config.Websites[k]
		r.GET(fmt.Sprintf("/%s", site.Identifier), func(context *gin.Context) {
			site.Lock.Lock()
			defer site.Lock.Unlock()

			if site.Result {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/status-up-success.svg")
			} else {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/status-down-critical.svg")
			}
		})
		r.PUT(fmt.Sprintf("/%s", site.Identifier), func(context *gin.Context) {
			if site.Mode != "Passive" {
				context.JSON(http.StatusMethodNotAllowed, gin.H{"error": "mode is not passive"})
			} else {
				site.Channel <- true
				context.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})
		r.GET(fmt.Sprintf("/%s-lastseen", site.Identifier), func(context *gin.Context) {
			site.Lock.Lock()
			defer site.Lock.Unlock()
			if !site.LastSeen.IsZero() {
				context.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("https://img.shields.io/badge/last seen-%s-blue.svg", site.LastSeen.Format("2006--01--02 15:04:05")))
			} else {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/last seen-n/a-blue.svg")
			}
		})
		r.GET(fmt.Sprintf("/%s-sla", site.Identifier), func(context *gin.Context) {
			site.Lock.Lock()
			defer site.Lock.Unlock()
			if site.ResultCount != 0 {
				var color string
				sla := float64(site.ResultSuccessCount) / float64(site.ResultCount)
				switch {
				case sla > 0.9:
					color = "green"
				case sla > 0.6:
					color = "yellow"
				default:
					color = "red"
				}
				context.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("https://img.shields.io/badge/sla-%.1f%%25-%s.svg", sla*100, color))
			} else {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/sla-n/a-blue.svg")
			}
		})
	}

	r.Run()
}
