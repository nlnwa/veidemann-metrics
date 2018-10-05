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
	r "gopkg.in/gorethink/gorethink.v4"
	"log"
)

// Connection holds the connections for Veidemann database
type Connection struct {
	DbConnectOpts r.ConnectOpts
	DbSession     r.QueryExecutor
}

// NewConnection creates a new Connection object
func NewConnection(dbHost string, dbPort int, dbUser string, dbPassword string, dbName string) *Connection {
	c := &Connection{
		DbConnectOpts: r.ConnectOpts{
			Address:    fmt.Sprintf("%s:%d", dbHost, dbPort),
			Username:   dbUser,
			Password:   dbPassword,
			Database:   dbName,
			NumRetries: 10,
		},
	}
	return c
}

// Connect establishes the databse connection
func (c *Connection) Connect() error {
	// Set up database connection
	if c.DbConnectOpts.Database == "mock" {
		c.DbSession = r.NewMock(c.DbConnectOpts)
	} else {
		dbSession, err := r.Connect(c.DbConnectOpts)
		if err != nil {
			log.Fatalf("fail to connect to database: %v", err)
			return err
		}
		c.DbSession = dbSession
	}

	log.Printf("Connected to DB at: %s", c.DbConnectOpts.Address)

	return nil
}
