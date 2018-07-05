package main

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

//Define a struct for urlStatus that contains pointers
//to prometheus descriptors for each metric you wish to expose.
type urlStatusCollector struct {
	urlMetric *prometheus.Desc
}

//initializes every descriptor and returns a pointer to the collector
func newurlStatusCollector(label []string) *urlStatusCollector {
	return &urlStatusCollector{
		urlMetric: prometheus.NewDesc("urlStatus",
			"Shows if a site is up",
			label, nil,
		),
	}
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel
func (collector *urlStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.urlMetric
}

//Collect implements required collect function for all promehteus collectors
func (collector *urlStatusCollector) Collect(ch chan<- prometheus.Metric) {
	//wait channel. used for waiting for all urls to report back with status.
	wait := make(chan string)
	//copy urls to local variable to prevent race condition
	localulrs := urls
	//loop through urls of all ingress objects and kick off a routine to check their status
	for _, url := range localulrs {
		go checkURL(collector, url, ch, wait)
	}
	//wait for all urls to respond.
	for i := 1; i <= len(localulrs); i++ {
		<-wait
	}
}

//Checks the url for each ingress object.
func checkURL(collector *urlStatusCollector, l string, ch chan<- prometheus.Metric, wait chan string) {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	r, err := client.Get(l) // does follow redirects
	var resp float64
	//borrowed cloudflares 523 error. usually this means the url is unreachable (networking/dns error)
	if r == nil {
		resp = 523
	} else {
		resp = float64(r.StatusCode)
	}
	//send http response code as value
	if err == nil {
		ch <- prometheus.MustNewConstMetric(collector.urlMetric, prometheus.CounterValue, resp, l)
		wait <- l
	} else {
		log.Warn(err)
		ch <- prometheus.MustNewConstMetric(collector.urlMetric, prometheus.CounterValue, resp, l)
		wait <- l
	}
}
