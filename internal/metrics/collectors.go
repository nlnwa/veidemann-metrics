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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "veidemann"
)

var (
	JobStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "job",
		Name:      "status_total",
		Help:      "Status for running jobs",
	}, []string{"job_name", "status"})

	JobSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "job",
		Name:      "size_total",
		Help:      "Sizes for running jobs",
	}, []string{"job_name", "type"})
)

func registerCollectors(collectUriQueueLength func() float64) {
	prometheus.MustRegister(version.NewCollector("veidemann_exporter"))

	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "uri",
			Name:      "queue_count",
			Help:      "Number of uris in queue.",
		}, func() float64 { return collectUriQueueLength() }))
}
