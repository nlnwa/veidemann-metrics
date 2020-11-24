package frontier

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nlnwa/veidemann-api-go/frontier/v1"
)

type Query interface {
	QueueCountTotal(ctx context.Context) (int64, error)
}

func (ac Client) QueueCountTotal(ctx context.Context) (int64, error) {
	client := frontier.NewFrontierClient(ac.conn)

	res, err := client.QueueCountTotal(ctx, &empty.Empty{})
	if err != nil {
		return 0, err
	}
	return res.GetCount(), nil
}
