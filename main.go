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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nlnwa/veidemann-metrics/internal/frontier"
	"github.com/nlnwa/veidemann-metrics/internal/logger"
	"github.com/nlnwa/veidemann-metrics/internal/metrics"
	"github.com/nlnwa/veidemann-metrics/internal/rethinkdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const indexContent = `<!DOCTYPE html>
<html lang="en">
<head><title>Veidemann Exporter</title></head>
<body>
<h1>Veidemann Exporter</h1>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>`

func main() {
	pflag.String("host", "", "Host")
	pflag.Int("port", 9301, "Port")

	pflag.String("db-host", "rethinkdb-proxy", "Database host")
	pflag.Int("db-port", 28015, "Database port")
	pflag.String("db-name", "veidemann", "Database name")
	pflag.String("db-username", "admin", "Database username")
	pflag.String("db-password", "", "Database password")

	pflag.String("frontier-host", "veidemann-frontier", "Frontier host")
	pflag.Int("frontier-port", 7700, "Frontier port")

	pflag.String("log-level", "info", "Log level; available levels are panic, fatal, error, warn, info, debug and trace")
	pflag.String("log-formatter", "logfmt", "Log formatter; available values are logfmt and json")
	pflag.Bool("log-method", false, "Log method names or not")

	pflag.Parse()

	_ = viper.BindPFlags(pflag.CommandLine)
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse flags")
	}

	logger.InitLog(viper.GetString("log-level"), viper.GetString("log-formatter"), viper.GetBool("log-method"))

	db := rethinkdb.NewConnection(
		viper.GetString("db-host"),
		viper.GetInt("db-port"),
		viper.GetString("db-username"),
		viper.GetString("db-password"),
		viper.GetString("db-name"),
		1*time.Minute)
	if err := db.Connect(); err != nil {
		log.Fatal().Err(err).
			Str("host", viper.GetString("db-host")).
			Int("port", viper.GetInt("db-port")).
			Msg("Failed to connect to RethinkDB")
	}
	defer func() { _ = db.Close() }()
	log.Info().
		Str("host", viper.GetString("db-host")).
		Int("port", viper.GetInt("db-port")).
		Msg("Connected to RethinkDB")

	if err := db.Verify(); err != nil {
		_ = db.Close()
		log.Fatal().Err(err).Msg("Database is not initialized")
	}

	frontierAddress := fmt.Sprintf("%s:%d", viper.GetString("frontier-host"), viper.GetInt("frontier-port"))
	conn, err := grpc.NewClient(frontierAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Str("address", frontierAddress).Msg("Failed to create frontier client")
	}
	defer conn.Close()

	log.Info().Str("address", frontierAddress).Msg("Frontier channel created")

	exp := metrics.New(db, frontier.New(conn))
	exp.Run(30 * time.Second)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(indexContent))
	})
	http.Handle("/metrics", promhttp.Handler())
	addr := fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port"))
	server := &http.Server{Addr: addr}

	go func() {
		signals := make(chan os.Signal, 2)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		sig := <-signals
		log.Info().Str("signal", sig.String()).Msg("Shutting down")
		_ = server.Close()
	}()

	// Serve metrics
	log.Info().Str("address", addr).Msg("Server listening")
	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Err(err).Msg("")
	}
}
