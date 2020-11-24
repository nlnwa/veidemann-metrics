/*
 * Copyright 2018 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package exporter

import (
	"context"
	"fmt"
	"github.com/nlnwa/veidemann-metrics/pkg/client/frontier"
	"github.com/nlnwa/veidemann-metrics/pkg/client/rethinkdb"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"strings"
	"time"
)

type exporter struct {
	rethinkdb rethinkdb.Query
	frontier  frontier.Query
}

// Exporter listens for changes to Veidemann database and exposes Prometheus metrics
type Exporter interface {
	Run()
}

// New creates a new Exporter
func New(rethinkdb rethinkdb.Query, frontier frontier.Query) Exporter {
	e := &exporter{
		rethinkdb,
		frontier,
	}
	e.registerCollectors()
	return e
}

func (e *exporter) Run() {
	crawlLog, err := e.rethinkdb.CrawlLogChanges()
	if err != nil {
		log.Fatal(err)
	}
	crawlLogChannel := make(chan map[string]interface{})
	go e.collectCrawlLog(crawlLogChannel)
	crawlLog.Listen(crawlLogChannel)

	pageLog, err := e.rethinkdb.PageLogChanges()
	if err != nil {
		log.Fatal(err)
	}
	pageLogChannel := make(chan map[string]interface{})
	go e.collectPageLog(pageLogChannel)
	pageLog.Listen(pageLogChannel)

	go e.collectJobStatusJob()
}

func (e *exporter) collectCrawlLog(ch chan map[string]interface{}) {
	for {
		response := <-ch
		if response == nil {
			panic("Connection closed")
		}

		newVal := response["new_val"].(map[string]interface{})

		collectors["uri.requests"].(prometheus.Counter).Inc()
		if newVal["error"] != nil {
			collectors["uri.requests.failed"].(prometheus.Counter).Inc()
		}
		collectors["uri.statuscode"].(*prometheus.CounterVec).WithLabelValues(fmt.Sprint(newVal["statusCode"])).Inc()
		collectors["uri.recordtype"].(*prometheus.CounterVec).WithLabelValues(fmt.Sprint(newVal["recordType"])).Inc()
		if mime, ok := getNormalizedMimeType(newVal); ok {
			collectors["uri.mime"].(*prometheus.CounterVec).WithLabelValues(mime).Inc()
		}
		if newVal["fetchTimeMs"] != nil {
			if fetchTimeMs, ok := newVal["fetchTimeMs"].(float64); ok {
				collectors["uri.fetchtime"].(prometheus.Summary).Observe(fetchTimeMs / 1000)
			}
		}
		if newVal["size"] != nil {
			if size, ok := newVal["size"].(float64); ok {
				collectors["uri.size"].(prometheus.Summary).Observe(size)
			}
		}
	}
}

func (e *exporter) collectPageLog(ch chan map[string]interface{}) {
	for {
		response := <-ch
		if response == nil {
			panic("Connection closed")
		}

		newVal := response["new_val"].(map[string]interface{})

		collectors["page.requests"].(prometheus.Counter).Inc()
		if newVal["outlink"] != nil {
			outlinks := newVal["outlink"].([]interface{})
			collectors["page.outlinks"].(prometheus.Summary).Observe(float64(len(outlinks)))
			collectors["page.links"].(*prometheus.CounterVec).WithLabelValues("outlinks").Add(float64(len(outlinks)))
		}
		if newVal["resource"] != nil {
			resources := newVal["resource"].([]interface{})
			var cached float64
			var notCached float64
			for _, resource := range resources {
				if resource.(map[string]interface{})["fromCache"] == true {
					cached++
				} else {
					notCached++
				}
			}
			collectors["page.resources"].(prometheus.Summary).Observe(cached + notCached)
			collectors["page.resources.cache.hit"].(prometheus.Summary).Observe(cached)
			collectors["page.resources.cache.miss"].(prometheus.Summary).Observe(notCached)
			collectors["page.links"].(*prometheus.CounterVec).WithLabelValues("resources_notcached").Add(notCached)
			collectors["page.links"].(*prometheus.CounterVec).WithLabelValues("resources_cached").Add(cached)
		}
	}
}

func (e *exporter) collectUriQueueLength() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	count, err := e.frontier.QueueCountTotal(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return float64(count)
}

func getNormalizedMimeType(doc map[string]interface{}) (string, bool) {
	if doc["contentType"] != nil {
		s := doc["contentType"].(string)
		i := strings.Index(s, ";")
		if i > 0 {
			return s[:i], true
		}
		return s, true
	}
	return "", false
}

func (e *exporter) collectJobStatusJob() {
	pollInterval := 120

	timerCh := time.Tick(time.Duration(pollInterval) * time.Second)

	e.collectJobStatus()
	for range timerCh {
		e.collectJobStatus()
	}
}

func (e *exporter) collectJobStatus() {
	jobStates, err := e.rethinkdb.JobStates()
	if err != nil {
		log.Fatal(err)
	}
	defer jobStates.Close()

	var jobState map[string]interface{}
	for jobStates.Next(&jobState) {
		c1 := collectors["job.status"].(*prometheus.GaugeVec)
		c1.WithLabelValues(jobState["name"].(string), "ABORTED_MANUAL").Set(jobState["ABORTED_MANUAL"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "ABORTED_SIZE").Set(jobState["ABORTED_SIZE"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "ABORTED_TIMEOUT").Set(jobState["ABORTED_TIMEOUT"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "CREATED").Set(jobState["CREATED"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "FAILED").Set(jobState["FAILED"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "FETCHING").Set(jobState["FETCHING"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "FINISHED").Set(jobState["FINISHED"].(float64))
		c1.WithLabelValues(jobState["name"].(string), "SLEEPING").Set(jobState["SLEEPING"].(float64))

		c2 := collectors["job.size"].(*prometheus.GaugeVec)
		c2.WithLabelValues(jobState["name"].(string), "documentsCrawled").Set(jobState["documentsCrawled"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "documentsDenied").Set(jobState["documentsDenied"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "documentsFailed").Set(jobState["documentsFailed"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "documentsOutOfScope").Set(jobState["documentsOutOfScope"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "documentsRetried").Set(jobState["documentsRetried"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "urisCrawled").Set(jobState["urisCrawled"].(float64))
		c2.WithLabelValues(jobState["name"].(string), "bytesCrawled").Set(jobState["bytesCrawled"].(float64))
	}
}
