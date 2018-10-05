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
	"flag"
)

const (
	defaultListenAddress = "127.0.0.1:9301"
	defaultMetricsPath   = "/metrics"
	defaultDbHost        = "localhost"
	defaultDbPort        = 28015
	defaultDbName        = "veidemann"
	defaultDbUser        = "admin"
	defaultDbPassword    = "admin"
)

/* Config configurations for exporter */
type Config struct {
	ListenAddress string
	ListenPort    int
	MetricPath    string

	DbHost     string
	DbPort     int
	DbUser     string
	DbPassword string
	DbName     string
}

/* NewConfig creates a new config object from command line args */
func NewConfig() *Config {
	c := &Config{}

	flag.StringVar(&c.ListenAddress, "listen", defaultListenAddress, "Address and Port to bind exporter, in host:port format")
	flag.StringVar(&c.MetricPath, "metrics-path", defaultMetricsPath, "Metrics path to expose prometheus metrics")

	flag.StringVar(&c.DbHost, "dbhostname", defaultDbHost, "DB hostname")
	flag.IntVar(&c.DbPort, "dbport", defaultDbPort, "DB port")
	flag.StringVar(&c.DbName, "dbname", defaultDbName, "DB schema name")
	flag.StringVar(&c.DbUser, "dbuser", defaultDbUser, "DB user name")
	flag.StringVar(&c.DbPassword, "dbpassword", defaultDbPassword, "DB password")

	flag.Parse()

	return c
}
