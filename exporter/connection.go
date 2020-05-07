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
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"sort"
	"time"
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
			Timeout:    10 * time.Minute,
		},
	}
	return c
}

// Connect establishes the databse connection
func (c *Connection) Connect(ctx context.Context) error {
	if err := c.doConnect(ctx); err != nil {
		return err
	}

	if err := c.checkDbExists(ctx); err != nil {
		return err
	}

	if err := c.checkTablesExists(ctx); err != nil {
		return err
	}

	log.Printf("Connected to DB at: %s", c.DbConnectOpts.Address)
	return nil
}

func (c *Connection) doConnect(ctx context.Context) error {
	// Set up database connection
	if c.DbConnectOpts.Database == "mock" {
		c.DbSession = r.NewMock(c.DbConnectOpts)
		return nil
	} else {
		var err error
		var dbSession *r.Session
		for {
			select {
			case <-ctx.Done():
				return err
			default:
				dbSession, err = r.Connect(c.DbConnectOpts)
				if err == nil {
					c.DbSession = dbSession
					return nil
				}
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (c *Connection) checkDbExists(ctx context.Context) error {
	var err error
	var cursor *r.Cursor
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("could not find database veidemann")
		default:
			if cursor, err = r.DBList().Run(c.DbSession); err == nil {
				var dbNames []string
				if err = cursor.All(&dbNames); err == nil {
					_ = cursor.Close()
					if contains(dbNames, "veidemann") {
						return nil
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Connection) checkTablesExists(ctx context.Context) error {
	var err error
	var cursor *r.Cursor
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("could not find database veidemann")
		default:
			if cursor, err = r.TableList().Run(c.DbSession); err == nil {
				var tableNames []string
				if err = cursor.All(&tableNames); err == nil {
					_ = cursor.Close()
					if contains(tableNames, "page_log", "crawl_log", "uri_queue", "config", "job_executions") {
						return nil
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func contains(list []string, item ...string) bool {
	sort.Strings(list)
	for _, s := range item {
		i := sort.SearchStrings(list, s)
		if i >= len(list) || list[i] != s {
			return false
		}
	}
	return true
}
