package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// MonitorWebsite sets a website to monitor
type MonitorWebsite struct {
	URL                string `json:"URL"`
	Identifier         string `json:"Identifier"`
	InsecureSkipVerify bool   `json:"InsecureSkipVerify"`

	Result   bool        `json:"-"`
	LastSeen time.Time   `json:"-"`
	Logger   *log.Entry  `json:"-"`
	Lock     *sync.Mutex `json:"-"`
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
			"URL":        site.URL,
			"Identifier": site.Identifier,
		})
		site.Logger.Info("monitor initialized")
	}
}

func test(site *MonitorWebsite) {
	timeout := time.Duration(time.Duration(config.Timeout) * time.Second)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: site.InsecureSkipVerify,
		},
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
			site.LastSeen = time.Now()
		} else {
			code := -1
			if resp != nil {
				code = resp.StatusCode
			}

			site.Logger.WithFields(log.Fields{
				"err":  err,
				"code": code,
			}).Trace("site is down")

			if site.Result {
				site.Logger.WithFields(log.Fields{
					"err":  err,
					"code": code,
				}).Info("site goes down!")
			}

			site.Result = false
		}

		site.Lock.Unlock()
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func main() {
	for k := range config.Websites {
		go test(&config.Websites[k])
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
		r.GET(fmt.Sprintf("/%s-lastseen", site.Identifier), func(context *gin.Context) {
			site.Lock.Lock()
			defer site.Lock.Unlock()
			if !site.LastSeen.IsZero() {
				context.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("https://img.shields.io/badge/last seen-%s-blue.svg", site.LastSeen.Format("2006--01--02 15:04:05")))
			} else {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/last seen-n/a-blue.svg")
			}
		})
	}

	r.Run()
}
