/*
 * Copyright 2020 National Library of Norway.
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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	frontierV1 "github.com/nlnwa/veidemann-api/go/frontier/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/rethinkdb/rethinkdb-go.v6/encoding"
)

var decodeJobExecutionStatus = func(encoded interface{}, value reflect.Value) error {
	b, err := json.Marshal(encoded)
	if err != nil {
		return fmt.Errorf("failed to marshal encoded value to json: %w", err)
	}

	var jes frontierV1.JobExecutionStatus
	err = protojson.UnmarshalOptions{AllowPartial: true}.Unmarshal(b, &jes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json to job execution status: %w", err)
	}

	value.Set(reflect.ValueOf(&jes).Elem())

	return nil
}

var encodeProtoMessage = func(value interface{}) (interface{}, error) {
	b, err := protojson.Marshal(value.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error decoding ConfigObject: %w", err)
	}

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("error encoding proto message: %w", err)
	}
	return encoding.Encode(m)
}

func init() {
	encoding.SetTypeEncoding(
		reflect.TypeOf(new(frontierV1.JobExecutionStatus)),
		encodeProtoMessage,
		decodeJobExecutionStatus,
	)
	encoding.SetTypeEncoding(
		reflect.TypeOf(map[string]interface{}{}),
		func(value interface{}) (i interface{}, err error) {
			m, ok := value.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("value is not a map[string]interface{}")
			}
			for k, v := range m {
				switch t := v.(type) {
				case string:
					// Try to parse string as date
					if ti, err := time.Parse(time.RFC3339Nano, t); err == nil {
						m[k] = ti
					} else {
						if m[k], err = encoding.Encode(v); err != nil {
							return nil, err
						}
					}
				default:
					if m[k], err = encoding.Encode(v); err != nil {
						return nil, err
					}
				}
			}
			return value, nil
		},
		func(encoded interface{}, value reflect.Value) error {
			value.Set(reflect.ValueOf(encoded))
			return nil
		},
	)
}
