package rethinkdb

import (
	"context"
	"fmt"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"sort"
)

type Query struct {
	*connection
}

// Verify verifies that database is initialized
func (qc *Query) Verify() error {
	if err := qc.checkDbExists(); err != nil {
		return err
	}
	if err := qc.checkTablesExists(); err != nil {
		return err
	}
	return nil
}

func (qc *Query) checkDbExists() error {
	cursor, err := r.DBList().Run(qc.session)
	if err != nil {
		return err
	}
	var dbList []string
	err = cursor.All(&dbList)
	if err != nil {
		return err
	}
	if !contains(dbList, "veidemann") {
		return fmt.Errorf("database 'veidemann' does not exist")
	}
	return nil
}

func (qc *Query) checkTablesExists() error {
	cursor, err := r.TableList().Run(qc.session)
	if err != nil {
		return err
	}
	var tableList []string
	err = cursor.All(&tableList)
	if err != nil {
		return err
	}
	if !contains(tableList, "config", "job_executions") {
		return fmt.Errorf("tables 'config' and 'job_executions' does not exist")
	}
	return nil
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

func (qc *Query) WalkLatestJobExecutionForCrawlJobs(ctx context.Context, fn func(*frontier.JobExecutionStatus)) error {
	cursor, err := r.Table("config").Filter(map[string]interface{}{"kind": "crawlJob"}).
		Map(func(job r.Term) interface{} {
			return r.Table("job_executions").
				OrderBy(r.OrderByOpts{Index: r.Desc("jobId_startTime")}).
				Between([]r.Term{job.Field("id"), r.MinVal}, []r.Term{job.Field("id"), r.MaxVal}).
				Limit(1).
				Map(func(jes r.Term) interface{} {
					return jes.Merge(map[string]interface{}{
						"executionsState": jes.Field("executionsState").
							ConcatMap(func(state r.Term) interface{} {
								return state.CoerceTo("array")
							}).
							CoerceTo("object"),
						"jobId": job.Field("meta").Field("name"),
					})
				}).
				Nth(0).
				Default(nil)
		}).
		Filter(func(jes r.Term) r.Term {
			return jes.Eq(nil).Not()
		}).
		Run(qc.session, r.RunOpts{
			Durability: "soft",
			ReadMode:   "outdated",
			Context:    ctx,
		})
	if err != nil {
		return err
	}

	jes := new(frontier.JobExecutionStatus)
	for cursor.Next(jes) {
		fn(jes)
	}
	return cursor.Err()
}
