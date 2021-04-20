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

package rethinkdb

import (
	"fmt"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"time"
)

// connection holds the connection to RethinkDB.
type connection struct {
	opts    r.ConnectOpts
	session *r.Session
}

func NewConnection(host string, port int, username string, password string, database string, timeout time.Duration) *Query {
	return &Query{
		connection: &connection{
			opts: r.ConnectOpts{
				Address:    fmt.Sprintf("%s:%d", host, port),
				Username:   username,
				Password:   password,
				Database:   database,
				NumRetries: 10,
				Timeout:    timeout,
			},
		},
	}
}

// Connect establishes the database connection.
func (c *connection) Connect() error {
	session, err := r.Connect(c.opts)
	if err != nil {
		return err
	}
	c.session = session
	return nil
}

// Close closes the database connection.
func (c *connection) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}
