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

type Collector struct {
	conn *Connection
}

func New(connection *Connection) *Collector {
	return &Collector{
		conn: connection,
	}
}

func (c *Collector) Run() {
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

func (c *Collector) collectCrawlLog(ch chan map[string]interface{}) {
	for {
		response := <-ch
		if response == nil {
			panic("Connection closed")
		}

		new_val := response["new_val"].(map[string]interface{})

		Collectors["uri.requests"].(prometheus.Counter).Inc()
		if new_val["error"] != nil {
			Collectors["uri.requests.failed"].(prometheus.Counter).Inc()
		}
		Collectors["uri.statuscode"].(*prometheus.CounterVec).WithLabelValues(fmt.Sprint(new_val["statusCode"])).Inc()
		Collectors["uri.recordtype"].(*prometheus.CounterVec).WithLabelValues(fmt.Sprint(new_val["recordType"])).Inc()
		if mime, ok := getNormalizedMimeType(new_val); ok {
			Collectors["uri.mime"].(*prometheus.CounterVec).WithLabelValues(mime).Inc()
		}
		if new_val["fetchTimeMs"] != nil {
			Collectors["uri.fetchtime"].(prometheus.Summary).Observe(new_val["fetchTimeMs"].(float64) / 1000)
		}
		if new_val["size"] != nil {
			Collectors["uri.size"].(prometheus.Summary).Observe(new_val["size"].(float64))
		}
	}
}

func (c *Collector) collectPageLog(ch chan map[string]interface{}) {
	for {
		response := <-ch
		if response == nil {
			panic("Connection closed")
		}

		new_val := response["new_val"].(map[string]interface{})

		Collectors["page.requests"].(prometheus.Counter).Inc()
		if new_val["outlink"] != nil {
			outlinks := new_val["outlink"].([]interface{})
			Collectors["page.outlinks"].(prometheus.Summary).Observe(float64(len(outlinks)))
		}
		if new_val["resource"] != nil {
			resources := new_val["resource"].([]interface{})
			var cached float64
			var notCached float64
			for _, resource := range resources {
				if resource.(map[string]interface{})["fromCache"] == true {
					cached++
				} else {
					notCached++
				}
			}
			Collectors["page.resources"].(prometheus.Summary).Observe(cached + notCached)
			Collectors["page.resources.cache.hit"].(prometheus.Summary).Observe(cached)
			Collectors["page.resources.cache.miss"].(prometheus.Summary).Observe(notCached)
		}
	}
}

func getNormalizedMimeType(doc map[string]interface{}) (string, bool) {
	if doc["contentType"] != nil {
		s := doc["contentType"].(string)
		i := strings.Index(s, ";")
		if i > 0 {
			return s[:i], true
		} else {
			return s, true
		}
	} else {
		return "", false
	}
}

func init() {
	prometheus.MustRegister(version.NewCollector("veidemann_exporter"))
	registerCollectors()
}
