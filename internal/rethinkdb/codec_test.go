package rethinkdb

import (
	"testing"

	frontierV1 "github.com/nlnwa/veidemann-api/go/frontier/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestUnmarshal(t *testing.T) {
	jesJSON := `{
"bytesCrawled": 4996378,
"documentsCrawled": 1,
"documentsOutOfScope": 64,
"documentsRetried": 2,
"endTime": "2021-04-20T13:31:07.952Z",
"executionsState": {
	"UNDEFINED":0,
	"CREATED":0,
	"FETCHING":0,
	"SLEEPING":0,
	"FINISHED":1,
	"ABORTED_TIMEOUT":0,
	"ABORTED_SIZE":0,
	"ABORTED_MANUAL":0,
	"FAILED":0,
	"DIED":0,
	"UNRECOGNIZED":0
},
"id": "e02ce980-eb0a-4573-ac52-cf9b695e5df5",
"jobId": "Unscheduled",
"startTime": "2021-04-20T13:08:55.651Z",
"state": "FINISHED",
"urisCrawled": 70
}`

	var jes frontierV1.JobExecutionStatus
	if err := (protojson.UnmarshalOptions{AllowPartial: true}).Unmarshal([]byte(jesJSON), &jes); err != nil {
		t.Errorf("failed to unmarshal json to job execution status: %v", err)
	}

}
