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
	r "gopkg.in/gorethink/gorethink.v4"
	"log"
	"strings"
	"time"
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
	e := &exporter{
		conn: connection,
	}
	e.registerCollectors()
	return e
}

func (e *exporter) Run() {
	crawlLog, err := r.Table("crawl_log").Changes().Run(e.conn.DbSession)
	if err != nil {
		log.Fatal(err)
	}
	crawlLogChannel := make(chan map[string]interface{})
	go e.collectCrawlLog(crawlLogChannel)
	crawlLog.Listen(crawlLogChannel)

	pageLog, err := r.Table("page_log").Changes().Run(e.conn.DbSession)
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
			collectors["uri.fetchtime"].(prometheus.Summary).Observe(newVal["fetchTimeMs"].(float64) / 1000)
		}
		if newVal["size"] != nil {
			collectors["uri.size"].(prometheus.Summary).Observe(newVal["size"].(float64))
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
	queueCount, err := r.Table("uri_queue").Count().Run(e.conn.DbSession)
	if err != nil {
		log.Fatal(err)
	}
	var result float64
	if err := queueCount.One(&result); err != nil {
		log.Fatal(err)
	}
	return result
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
	jobStates, err := r.Table("config").Filter(map[string]interface{}{"kind": "crawlJob"}).
		Map(func(d r.Term) interface{} {
			return r.Table("job_executions").
				OrderBy(r.OrderByOpts{Index: r.Desc("jobId_startTime")}).
				Between([]r.Term{d.Field("id"), r.MinVal}, []r.Term{d.Field("id"), r.MaxVal}).
				Limit(1).
				Map(func(doc r.Term) interface{} {
					return doc.Field("executionsState").
						Do(func(d2 r.Term) interface{} {
							return d2.ConcatMap(func(d3 r.Term) interface{} {
								return d3.CoerceTo("array")
							}).CoerceTo("object")
						}).Default(map[string]interface{}{}).
						Merge(
							map[string]interface{}{
								"documentsCrawled":    doc.Field("documentsCrawled").Default(0),
								"documentsDenied":     doc.Field("documentsDenied").Default(0),
								"documentsFailed":     doc.Field("documentsFailed").Default(0),
								"documentsOutOfScope": doc.Field("documentsOutOfScope").Default(0),
								"documentsRetried":    doc.Field("documentsRetried").Default(0),
								"urisCrawled":         doc.Field("urisCrawled").Default(0),
								"bytesCrawled":        doc.Field("bytesCrawled").Default(0),
								"state":               doc.Field("state").Default("UNDEFINED"),
								"jobExecutionId":      doc.Field("id").Default(""),
							})
				}).Nth(0).
				//Reduce(func(left, right r.Term) interface{} { return left }).
				Default(map[string]interface{}{
					"ABORTED_MANUAL":      0,
					"ABORTED_SIZE":        0,
					"ABORTED_TIMEOUT":     0,
					"CREATED":             0,
					"FAILED":              0,
					"FETCHING":            0,
					"FINISHED":            0,
					"SLEEPING":            0,
					"documentsCrawled":    0,
					"documentsDenied":     0,
					"documentsFailed":     0,
					"documentsOutOfScope": 0,
					"documentsRetried":    0,
					"urisCrawled":         0,
					"bytesCrawled":        0,
					"state":               "UNDEFINED",
				}).
				Merge(map[string]interface{}{"name": d.Field("meta").Field("name"),}).
				Do(func(doc r.Term) interface{} {
					return r.Branch(doc.Field("state").Eq("RUNNING"),
						doc.Merge(func(d r.Term) interface{} {
							return r.Table("executions").
								Between([]r.Term{d.Field("jobExecutionId"), r.MinVal}, []r.Term{d.Field("jobExecutionId"), r.MaxVal}, r.BetweenOpts{
									Index: "jobExecutionId_seedId",
								}).
								Map(func(doc r.Term) interface{} {
									return map[string]interface{}{
										"ABORTED_MANUAL":      r.Branch(doc.Field("state").Eq("ABORTED_MANUAL"), 1, 0),
										"ABORTED_SIZE":        r.Branch(doc.Field("state").Eq("ABORTED_SIZE"), 1, 0),
										"ABORTED_TIMEOUT":     r.Branch(doc.Field("state").Eq("ABORTED_TIMEOUT"), 1, 0),
										"CREATED":             r.Branch(doc.Field("state").Eq("CREATED"), 1, 0),
										"FAILED":              r.Branch(doc.Field("state").Eq("FAILED"), 1, 0),
										"FETCHING":            r.Branch(doc.Field("state").Eq("FETCHING"), 1, 0),
										"FINISHED":            r.Branch(doc.Field("state").Eq("FINISHED"), 1, 0),
										"SLEEPING":            r.Branch(doc.Field("state").Eq("SLEEPING"), 1, 0),
										"documentsCrawled":    doc.Field("documentsCrawled").Default(0),
										"documentsDenied":     doc.Field("documentsDenied").Default(0),
										"documentsFailed":     doc.Field("documentsFailed").Default(0),
										"documentsOutOfScope": doc.Field("documentsOutOfScope").Default(0),
										"documentsRetried":    doc.Field("documentsRetried").Default(0),
										"urisCrawled":         doc.Field("urisCrawled").Default(0),
										"bytesCrawled":        doc.Field("bytesCrawled").Default(0),
									}
								}).Reduce(func(left, right r.Term) interface{} {
								return map[string]interface{}{
									"ABORTED_MANUAL":      left.Field("ABORTED_MANUAL").Add(right.Field("ABORTED_MANUAL")),
									"ABORTED_SIZE":        left.Field("ABORTED_SIZE").Add(right.Field("ABORTED_SIZE")),
									"ABORTED_TIMEOUT":     left.Field("ABORTED_TIMEOUT").Add(right.Field("ABORTED_TIMEOUT")),
									"CREATED":             left.Field("CREATED").Add(right.Field("CREATED")),
									"FAILED":              left.Field("FAILED").Add(right.Field("FAILED")),
									"FETCHING":            left.Field("FETCHING").Add(right.Field("FETCHING")),
									"FINISHED":            left.Field("FINISHED").Add(right.Field("FINISHED")),
									"SLEEPING":            left.Field("SLEEPING").Add(right.Field("SLEEPING")),
									"documentsCrawled":    left.Field("documentsCrawled").Add(right.Field("documentsCrawled")),
									"documentsDenied":     left.Field("documentsDenied").Add(right.Field("documentsDenied")),
									"documentsFailed":     left.Field("documentsFailed").Add(right.Field("documentsFailed")),
									"documentsOutOfScope": left.Field("documentsOutOfScope").Add(right.Field("documentsOutOfScope")),
									"documentsRetried":    left.Field("documentsRetried").Add(right.Field("documentsRetried")),
									"urisCrawled":         left.Field("urisCrawled").Add(right.Field("urisCrawled")),
									"bytesCrawled":        left.Field("bytesCrawled").Add(right.Field("bytesCrawled")),
								}
							})
						}),
						doc)
				})
		}).
		Run(e.conn.DbSession)

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
