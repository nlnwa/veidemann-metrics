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

package main

import (
	"context"
	"fmt"
	"github.com/nlnwa/veidemann-metrics/pkg/client/frontier"
	"github.com/nlnwa/veidemann-metrics/pkg/client/rethinkdb"
	"github.com/nlnwa/veidemann-metrics/pkg/exporter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"
)

const indexContent = `<html>
             <head><title>Veidemann Exporter</title></head>
             <body>
             <h1>Veidemann Exporter</h1>
             <p><a href='` + "/metrics" + `'>Metrics</a></p>
             </body>
             </html>
`

func main() {
	config := NewConfig()
	db := rethinkdb.NewConnection(config.DbHost, config.DbPort, config.DbUser, config.DbPassword, config.DbName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := db.Connect(ctx); err != nil {
		log.Fatal(err)
	}

	fCtx, fCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer fCancel()
	f := frontier.New(config.FrontierHost, config.FrontierPort)
	if err := f.Connect(fCtx); err != nil {
		log.Fatal(err)
	}
	exp := exporter.New(db, f)
	exp.Run()

	// Serve metrics
	http.Handle(config.MetricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexContent))
	})

	log.Println("Listening on", fmt.Sprintf("%s", config.ListenAddress))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", config.ListenAddress), nil))
}
