package ltops

import (
	"io"
)

// LoadTestOptions defines the possible options when running a Mattermost load test.
type LoadTestOptions struct {
	ConfigFile    string
	Workers       int // how many workers to execute in parallel the bulk import
	SkipBulkLoad  bool
	ForceBulkLoad bool // force bulk load even if previously loaded
	ResultsWriter io.Writer
}
