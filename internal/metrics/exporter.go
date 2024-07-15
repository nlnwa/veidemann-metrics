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

package metrics

import (
	"context"
	frontierV1 "github.com/nlnwa/veidemann-api/go/frontier/v1"
	"github.com/nlnwa/veidemann-metrics/internal/frontier"
	"github.com/nlnwa/veidemann-metrics/internal/rethinkdb"
	"log"
	"time"
)

type Exporter struct {
	rethinkdb *rethinkdb.Query
	frontier  *frontier.Client
}

// New creates a new Exporter
func New(rethinkdb *rethinkdb.Query, frontier *frontier.Client) *Exporter {
	return &Exporter{
		rethinkdb,
		frontier,
	}
}

func (e *Exporter) Run(interval time.Duration) {
	registerCollectors(e.collectUriQueueLength)
	go func() {
		e.collectJobStatusJob()
		for range time.Tick(interval) {
			e.collectJobStatusJob()
		}
	}()
}

func (e *Exporter) collectJobStatusJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := e.rethinkdb.WalkLatestJobExecutionForCrawlJobs(ctx, collectJobStatus)
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Exporter) collectUriQueueLength() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	count, err := e.frontier.QueueCountTotal(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return float64(count)
}

func collectJobStatus(jobState *frontierV1.JobExecutionStatus) {
	name := jobState.GetJobId()
	stateOrDefault := getOrDefault(jobState.GetExecutionsState())
	JobStatus.WithLabelValues(name, "ABORTED_MANUAL").Set(stateOrDefault("ABORTED_MANUAL", 0))
	JobStatus.WithLabelValues(name, "ABORTED_SIZE").Set(stateOrDefault("ABORTED_SIZE", 0))
	JobStatus.WithLabelValues(name, "ABORTED_TIMEOUT").Set(stateOrDefault("ABORTED_TIMEOUT", 0))
	JobStatus.WithLabelValues(name, "CREATED").Set(stateOrDefault("CREATED", 0))
	JobStatus.WithLabelValues(name, "FAILED").Set(stateOrDefault("FAILED", 0))
	JobStatus.WithLabelValues(name, "FETCHING").Set(stateOrDefault("FETCHING", 0))
	JobStatus.WithLabelValues(name, "FINISHED").Set(stateOrDefault("FINISHED", 0))
	JobStatus.WithLabelValues(name, "SLEEPING").Set(stateOrDefault("SLEEPING", 0))

	JobSize.WithLabelValues(name, "documentsCrawled").Set(float64(jobState.GetDocumentsCrawled()))
	JobSize.WithLabelValues(name, "documentsDenied").Set(float64(jobState.GetDocumentsDenied()))
	JobSize.WithLabelValues(name, "documentsFailed").Set(float64(jobState.GetDocumentsFailed()))
	JobSize.WithLabelValues(name, "documentsOutOfScope").Set(float64(jobState.GetDocumentsOutOfScope()))
	JobSize.WithLabelValues(name, "documentsRetried").Set(float64(jobState.GetDocumentsRetried()))
	JobSize.WithLabelValues(name, "urisCrawled").Set(float64(jobState.GetUrisCrawled()))
	JobSize.WithLabelValues(name, "bytesCrawled").Set(float64(jobState.GetBytesCrawled()))
}

func getOrDefault(m map[string]int32) func(k string, v int32) float64 {
	return func(k string, v int32) float64 {
		if value, ok := m[k]; !ok {
			return float64(v)
		} else {
			return float64(value)
		}
	}
}
