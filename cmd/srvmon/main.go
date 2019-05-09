package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// MonitorWebsite sets a website to monitor
type MonitorWebsite struct {
	URL        string     `json:"URL"`
	Identifier string     `json:"Identifier"`
	Logger     *log.Entry `json:"-"`
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

var results map[string]bool
var resultLocks map[string]*sync.Mutex

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

	results = make(map[string]bool)
	resultLocks = make(map[string]*sync.Mutex)

	for i, site := range config.Websites {
		results[site.Identifier] = false
		resultLocks[site.Identifier] = &sync.Mutex{}
		config.Websites[i].Logger = log.WithFields(log.Fields{
			"URL":        site.URL,
			"Identifier": site.Identifier,
		})
		config.Websites[i].Logger.Info("monitor initialized")
	}
}

func test(site MonitorWebsite) {
	site.Logger.Trace("testing connectivity...")
	timeout := time.Duration(time.Duration(config.Timeout) * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(site.URL)
	resultLocks[site.Identifier].Lock()
	if err == nil && resp.StatusCode == 200 {
		site.Logger.Trace("site is up")
		if !results[site.Identifier] {
			site.Logger.Info("site restored.")
		}
		results[site.Identifier] = true
	} else {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}
		site.Logger.WithFields(log.Fields{
			"err":  err,
			"code": code,
		}).Trace("site is down")
		if results[site.Identifier] {
			site.Logger.WithFields(log.Fields{
				"err":  err,
				"code": code,
			}).Info("site goes down!")
		}
		results[site.Identifier] = false
	}
	resultLocks[site.Identifier].Unlock()
}

func testAllWebsitesPeriodically() {
	for {
		for _, site := range config.Websites {
			go test(site)
		}
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func main() {
	go testAllWebsitesPeriodically()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/", func(context *gin.Context) {
		context.JSON(200, map[string]string{
			"running": "yes",
		})
	})

	for k := range config.Websites {
		id := config.Websites[k].Identifier
		r.GET("/"+id, func(context *gin.Context) {
			resultLocks[id].Lock()
			defer resultLocks[id].Unlock()
			if results[id] {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/status-up-success.svg")
			} else {
				context.Redirect(http.StatusTemporaryRedirect, "https://img.shields.io/badge/status-down-critical.svg")
			}
		})
	}

	r.Run()
}
