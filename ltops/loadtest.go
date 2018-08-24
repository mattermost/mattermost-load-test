package ltops

import "io"

// LoadTestOptions defines the possible options when starting a Mattermost load test.
type LoadTestOptions struct {
	Workers       int       // how many workers to execute in parallel the bulk import
	SkipBulkLoad  bool      // skip bulk load
	ForceBulkLoad bool      // force bulk load even if previously loaded
	ResultsWriter io.Writer // writer to write the results to
}
