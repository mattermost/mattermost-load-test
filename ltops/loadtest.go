package ltops

import "io"

// LoadTestOptions defines the possible options when starting a Mattermost load test.
type LoadTestOptions struct {
	ForceBulkLoad bool      // force bulk load even if previously loaded
	ResultsWriter io.Writer // writer to write the results to
}
