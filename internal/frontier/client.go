// Copyright 2018 National Library of Norway
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package aggregator contains an aggregator service client
package frontier

import (
	"context"
	"fmt"

	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	"google.golang.org/grpc"
)

type grpcClient struct {
	address string
	*grpc.ClientConn
}

func (ac *grpcClient) Connect(ctx context.Context) error {
	conn, err := grpc.DialContext(ctx, ac.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", ac.address, err)
	}
	ac.ClientConn = conn
	return nil
}

func (ac *grpcClient) Close() {
	if ac.ClientConn != nil {
		_ = ac.ClientConn.Close()
	}
}

type Client struct {
	*grpcClient
	frontier.FrontierClient
}

func New(host string, port int) *Client {
	return &Client{
		grpcClient: &grpcClient{
			address: fmt.Sprintf("%s:%d", host, port),
		},
	}
}

func (f *Client) Connect(ctx context.Context) error {
	err := f.grpcClient.Connect(ctx)
	if err != nil {
		return err
	}
	f.FrontierClient = frontier.NewFrontierClient(f.ClientConn)
	return nil
}

func (f *Client) QueueCountTotal(ctx context.Context) (int64, error) {
	res, err := f.FrontierClient.QueueCountTotal(ctx, &emptypb.Empty{})
	if err != nil {
		return 0, err
	}
	return res.GetCount(), nil
}
