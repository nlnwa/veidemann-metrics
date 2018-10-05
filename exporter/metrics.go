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
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	r "gopkg.in/gorethink/gorethink.v4"
	"log"
	"strings"
)

type exporter struct {
	conn *Connection
}

// Exporter listens for changes to Veidemann database and exposes Prometheus metrics
type Exporter interface {
	Run()
}

// New creates a new Exporter
func New(connection *Connection) Exporter {
	return &exporter{
		conn: connection,
	}
}

func (c *exporter) Run() {
	crawlLog, err := r.Table("crawl_log").Changes().Run(c.conn.DbSession)
	if err != nil {
		log.Fatal(err)
	}
	crawlLogChannel := make(chan map[string]interface{})
	go c.collectCrawlLog(crawlLogChannel)
	crawlLog.Listen(crawlLogChannel)

	pageLog, err := r.Table("page_log").Changes().Run(c.conn.DbSession)
	if err != nil {
		log.Fatal(err)
	}
	pageLogChannel := make(chan map[string]interface{})
	go c.collectPageLog(pageLogChannel)
	pageLog.Listen(pageLogChannel)
}

func (c *exporter) collectCrawlLog(ch chan map[string]interface{}) {
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
			collectors["uri.fetchtime"].(prometheus.Summary).Observe(newVal["fetchTimeMs"].(float64) / 1000)
		}
		if newVal["size"] != nil {
			collectors["uri.size"].(prometheus.Summary).Observe(newVal["size"].(float64))
		}
	}
}

func (c *exporter) collectPageLog(ch chan map[string]interface{}) {
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
		}
	}
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

func init() {
	prometheus.MustRegister(version.NewCollector("veidemann_exporter"))
	registerCollectors()
}
