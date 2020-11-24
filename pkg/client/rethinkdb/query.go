package rethinkdb

import r "gopkg.in/rethinkdb/rethinkdb-go.v6"

type Query interface {
	CrawlLogChanges() (*r.Cursor, error)
	PageLogChanges() (*r.Cursor, error)
	JobStates() (*r.Cursor, error)
}

func (c *Connection) CrawlLogChanges() (*r.Cursor, error) {
	return r.Table("crawl_log").Changes().Run(c.DbSession)
}

func (c *Connection) PageLogChanges() (*r.Cursor, error) {
	return r.Table("page_log").Changes().Run(c.DbSession)
}

func (c *Connection) JobStates() (*r.Cursor, error) {
	return r.Table("config").Filter(map[string]interface{}{"kind": "crawlJob"}).
		Map(func(d r.Term) interface{} {
			return r.Table("job_executions").
				OrderBy(r.OrderByOpts{Index: r.Desc("jobId_startTime")}).
				Between([]r.Term{d.Field("id"), r.MinVal}, []r.Term{d.Field("id"), r.MaxVal}).
				Limit(1).
				Map(func(doc r.Term) interface{} {
					return doc.Field("executionsState").
						Do(func(d2 r.Term) interface{} {
							return d2.ConcatMap(func(d3 r.Term) interface{} {
								return d3.CoerceTo("array")
							}).CoerceTo("object")
						}).Default(map[string]interface{}{}).
						Merge(
							map[string]interface{}{
								"documentsCrawled":    doc.Field("documentsCrawled").Default(0),
								"documentsDenied":     doc.Field("documentsDenied").Default(0),
								"documentsFailed":     doc.Field("documentsFailed").Default(0),
								"documentsOutOfScope": doc.Field("documentsOutOfScope").Default(0),
								"documentsRetried":    doc.Field("documentsRetried").Default(0),
								"urisCrawled":         doc.Field("urisCrawled").Default(0),
								"bytesCrawled":        doc.Field("bytesCrawled").Default(0),
								"state":               doc.Field("state").Default("UNDEFINED"),
								"jobExecutionId":      doc.Field("id").Default(""),
							})
				}).Nth(0).
				//Reduce(func(left, right r.Term) interface{} { return left }).
				Default(map[string]interface{}{
					"ABORTED_MANUAL":      0,
					"ABORTED_SIZE":        0,
					"ABORTED_TIMEOUT":     0,
					"CREATED":             0,
					"FAILED":              0,
					"FETCHING":            0,
					"FINISHED":            0,
					"SLEEPING":            0,
					"documentsCrawled":    0,
					"documentsDenied":     0,
					"documentsFailed":     0,
					"documentsOutOfScope": 0,
					"documentsRetried":    0,
					"urisCrawled":         0,
					"bytesCrawled":        0,
					"state":               "UNDEFINED",
				}).
				Merge(map[string]interface{}{"name": d.Field("meta").Field("name")}).
				Do(func(doc r.Term) interface{} {
					return r.Branch(r.Not(doc.HasFields("CREATED")),
						doc.Merge(func(d r.Term) interface{} {
							return r.Table("executions").
								Between([]r.Term{d.Field("jobExecutionId"), r.MinVal}, []r.Term{d.Field("jobExecutionId"), r.MaxVal}, r.BetweenOpts{
									Index: "jobExecutionId_seedId",
								}).
								Map(func(doc r.Term) interface{} {
									return map[string]interface{}{
										"ABORTED_MANUAL":      r.Branch(doc.Field("state").Eq("ABORTED_MANUAL"), 1, 0),
										"ABORTED_SIZE":        r.Branch(doc.Field("state").Eq("ABORTED_SIZE"), 1, 0),
										"ABORTED_TIMEOUT":     r.Branch(doc.Field("state").Eq("ABORTED_TIMEOUT"), 1, 0),
										"CREATED":             r.Branch(doc.Field("state").Eq("CREATED"), 1, 0),
										"FAILED":              r.Branch(doc.Field("state").Eq("FAILED"), 1, 0),
										"FETCHING":            r.Branch(doc.Field("state").Eq("FETCHING"), 1, 0),
										"FINISHED":            r.Branch(doc.Field("state").Eq("FINISHED"), 1, 0),
										"SLEEPING":            r.Branch(doc.Field("state").Eq("SLEEPING"), 1, 0),
										"documentsCrawled":    doc.Field("documentsCrawled").Default(0),
										"documentsDenied":     doc.Field("documentsDenied").Default(0),
										"documentsFailed":     doc.Field("documentsFailed").Default(0),
										"documentsOutOfScope": doc.Field("documentsOutOfScope").Default(0),
										"documentsRetried":    doc.Field("documentsRetried").Default(0),
										"urisCrawled":         doc.Field("urisCrawled").Default(0),
										"bytesCrawled":        doc.Field("bytesCrawled").Default(0),
									}
								}).
								Reduce(func(left, right r.Term) interface{} {
									return map[string]interface{}{
										"ABORTED_MANUAL":      left.Field("ABORTED_MANUAL").Add(right.Field("ABORTED_MANUAL")),
										"ABORTED_SIZE":        left.Field("ABORTED_SIZE").Add(right.Field("ABORTED_SIZE")),
										"ABORTED_TIMEOUT":     left.Field("ABORTED_TIMEOUT").Add(right.Field("ABORTED_TIMEOUT")),
										"CREATED":             left.Field("CREATED").Add(right.Field("CREATED")),
										"FAILED":              left.Field("FAILED").Add(right.Field("FAILED")),
										"FETCHING":            left.Field("FETCHING").Add(right.Field("FETCHING")),
										"FINISHED":            left.Field("FINISHED").Add(right.Field("FINISHED")),
										"SLEEPING":            left.Field("SLEEPING").Add(right.Field("SLEEPING")),
										"documentsCrawled":    left.Field("documentsCrawled").Add(right.Field("documentsCrawled")),
										"documentsDenied":     left.Field("documentsDenied").Add(right.Field("documentsDenied")),
										"documentsFailed":     left.Field("documentsFailed").Add(right.Field("documentsFailed")),
										"documentsOutOfScope": left.Field("documentsOutOfScope").Add(right.Field("documentsOutOfScope")),
										"documentsRetried":    left.Field("documentsRetried").Add(right.Field("documentsRetried")),
										"urisCrawled":         left.Field("urisCrawled").Add(right.Field("urisCrawled")),
										"bytesCrawled":        left.Field("bytesCrawled").Add(right.Field("bytesCrawled")),
									}
								}).
								Default(map[string]interface{}{
									"ABORTED_MANUAL":      0,
									"ABORTED_SIZE":        0,
									"ABORTED_TIMEOUT":     0,
									"CREATED":             0,
									"FAILED":              0,
									"FETCHING":            0,
									"FINISHED":            0,
									"SLEEPING":            0,
									"documentsCrawled":    0,
									"documentsDenied":     0,
									"documentsFailed":     0,
									"documentsOutOfScope": 0,
									"documentsRetried":    0,
									"urisCrawled":         0,
									"bytesCrawled":        0,
								})
						}),
						doc)
				})
		}).
		Run(c.DbSession)
}
