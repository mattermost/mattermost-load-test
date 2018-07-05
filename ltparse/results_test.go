package ltparse_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-load-test/loadtest"
	"github.com/mattermost/mattermost-load-test/ltparse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodeClientTimingStats(timings *loadtest.ClientTimingStats) string {
	if timings == nil {
		return `{"tag":"timings","timings":{}}`
	}

	timingsEncoded, _ := json.Marshal(timings)

	return fmt.Sprintf(`{"tag":"timings","timings":%s}`, string(timingsEncoded))
}

func TestParseTextResults(t *testing.T) {
	testCases := []struct {
		Description string
		Input       string

		ExpectedError         bool
		ExpectedOutput        string
		ExpectedVerboseOutput string
	}{
		{
			"malformed input",
			"{not json",

			true,
			"",
			"",
		},
		{
			"no routes",
			encodeClientTimingStats(nil),

			false,
			`--------- Timings Report ------------
Score: 0.00
`,
			`--------- Timings Report ------------
Score: 0.00
`,
		},
		{
			"route with no data points",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name: "/test/route/1",
						},
					},
				},
			),

			false,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 0
Error Rate: 0.00%
Mean Response Time: 0.00ms
Median Response Time: 0.00ms
95th Percentile: 0.00ms

Score: 0.00
`,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 0
Error Rate: 0.00%
Mean Response Time: 0.00ms
Median Response Time: 0.00ms
95th Percentile: 0.00ms
90th Percentile: 0.00ms
Max Response Time: 0ms
Min Response Time: 0ms
Inter Quartile Range: 0

Score: 0.00
`,
		},
		{
			"route with data points, no errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:    "/test/route/1",
							NumHits: 15,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
							},
						},
					},
				},
			),

			false,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 15
Error Rate: 0.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms

Score: 134.00
`,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 15
Error Rate: 0.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms
90th Percentile: 70.00ms
Max Response Time: 100ms
Min Response Time: 1ms
Inter Quartile Range: 36

Score: 134.00
`,
		},
		{
			"route with data points, some errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   20,
							NumErrors: 5,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
							},
						},
					},
				},
			),

			false,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 20
Error Rate: 25.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms

Score: 134.00
`,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 20
Error Rate: 25.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms
90th Percentile: 70.00ms
Max Response Time: 100ms
Min Response Time: 1ms
Inter Quartile Range: 36

Score: 134.00
`,
		},
		{
			"route without data points, all errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   20,
							NumErrors: 20,
							Duration:  []float64{},
						},
					},
				},
			),

			false,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 20
Error Rate: 100.00%
Mean Response Time: 0.00ms
Median Response Time: 0.00ms
95th Percentile: 0.00ms

Score: 0.00
`,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 20
Error Rate: 100.00%
Mean Response Time: 0.00ms
Median Response Time: 0.00ms
95th Percentile: 0.00ms
90th Percentile: 0.00ms
Max Response Time: 0ms
Min Response Time: 0ms
Inter Quartile Range: 0

Score: 0.00
`,
		},
		{
			"multiple routes",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   25,
							NumErrors: 10,
							Duration: []float64{
								20, 40, 60, 80, 100,
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
							},
						},
						"/test/route/2": &loadtest.RouteStats{
							Name:      "/test/route/2",
							NumHits:   16,
							NumErrors: 0,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
								1000,
							},
						},
					},
				},
			),

			false,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 25
Error Rate: 40.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms

Route: /test/route/2
Total Hits: 16
Error Rate: 0.00%
Mean Response Time: 84.69ms
Median Response Time: 8.50ms
95th Percentile: 550.00ms

Score: 369.00
`,
			`--------- Timings Report ------------
Route: /test/route/1
Total Hits: 25
Error Rate: 40.00%
Mean Response Time: 23.67ms
Median Response Time: 8.00ms
95th Percentile: 90.00ms
90th Percentile: 70.00ms
Max Response Time: 100ms
Min Response Time: 1ms
Inter Quartile Range: 36

Route: /test/route/2
Total Hits: 16
Error Rate: 0.00%
Mean Response Time: 84.69ms
Median Response Time: 8.50ms
95th Percentile: 550.00ms
90th Percentile: 90.00ms
Max Response Time: 1000ms
Min Response Time: 1ms
Inter Quartile Range: 45.5

Score: 369.00
`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			test := func(t *testing.T, verbose bool, expectedOutput string) {
				output := &strings.Builder{}
				config := &ltparse.ResultsConfig{
					Input:         strings.NewReader(testCase.Input),
					BaselineInput: nil,
					Output:        output,
					Display:       "text",
					Aggregate:     false,
					Verbose:       verbose,
				}

				err := ltparse.ParseResults(config)
				if testCase.ExpectedError {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, expectedOutput, output.String())
			}

			t.Run("normal", func(t *testing.T) {
				test(t, false, testCase.ExpectedOutput)
			})

			t.Run("verbose", func(t *testing.T) {
				test(t, true, testCase.ExpectedVerboseOutput)
			})
		})
	}
}

func TestParseMarkdownResults(t *testing.T) {
	testCases := []struct {
		Description string
		Input       string

		ExpectedError         bool
		ExpectedOutput        string
		ExpectedVerboseOutput string
	}{
		{
			"malformed input",
			"{not json",

			true,
			"",
			"",
		},
		{
			"no routes",
			encodeClientTimingStats(nil),

			false,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
`,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
`,
		},
		{
			"route with no data points",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name: "/test/route/1",
						},
					},
				},
			),

			false,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 0 |
| Error Rate | 0.00% |
| Mean Response Time | 0.00ms |
| Median Response Time | 0.00ms |
| 95th Percentile | 0.00ms |

`,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 0 |
| Error Rate | 0.00% |
| Mean Response Time | 0.00ms |
| Median Response Time | 0.00ms |
| 95th Percentile | 0.00ms |
| 90th Percentile | 0.00ms |
| Max Response Time | 0ms |
| Min Response Time | 0ms |
| Inter Quartile Range | 0 |

`,
		},
		{
			"route with data points, no errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:    "/test/route/1",
							NumHits: 15,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
							},
						},
					},
				},
			),

			false,
			`## Loadtest Results
### Score: 134.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 15 |
| Error Rate | 0.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |

`,
			`## Loadtest Results
### Score: 134.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 15 |
| Error Rate | 0.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |
| 90th Percentile | 70.00ms |
| Max Response Time | 100ms |
| Min Response Time | 1ms |
| Inter Quartile Range | 36 |

`,
		},
		{
			"route with data points, some errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   20,
							NumErrors: 5,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
							},
						},
					},
				},
			),

			false,
			`## Loadtest Results
### Score: 134.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 20 |
| Error Rate | 25.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |

`,
			`## Loadtest Results
### Score: 134.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 20 |
| Error Rate | 25.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |
| 90th Percentile | 70.00ms |
| Max Response Time | 100ms |
| Min Response Time | 1ms |
| Inter Quartile Range | 36 |

`,
		},
		{
			"route without data points, all errors",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   20,
							NumErrors: 20,
							Duration:  []float64{},
						},
					},
				},
			),

			false,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 20 |
| Error Rate | 100.00% |
| Mean Response Time | 0.00ms |
| Median Response Time | 0.00ms |
| 95th Percentile | 0.00ms |

`,
			`## Loadtest Results
### Score: 0.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 20 |
| Error Rate | 100.00% |
| Mean Response Time | 0.00ms |
| Median Response Time | 0.00ms |
| 95th Percentile | 0.00ms |
| 90th Percentile | 0.00ms |
| Max Response Time | 0ms |
| Min Response Time | 0ms |
| Inter Quartile Range | 0 |

`,
		},
		{
			"multiple routes",
			encodeClientTimingStats(
				&loadtest.ClientTimingStats{
					Routes: map[string]*loadtest.RouteStats{
						"/test/route/1": &loadtest.RouteStats{
							Name:      "/test/route/1",
							NumHits:   25,
							NumErrors: 10,
							Duration: []float64{
								20, 40, 60, 80, 100,
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
							},
						},
						"/test/route/2": &loadtest.RouteStats{
							Name:      "/test/route/2",
							NumHits:   16,
							NumErrors: 0,
							Duration: []float64{
								1, 2, 3, 4, 5,
								6, 7, 8, 9, 10,
								20, 40, 60, 80, 100,
								1000,
							},
						},
					},
				},
			),

			false,
			`## Loadtest Results
### Score: 369.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 25 |
| Error Rate | 40.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |

#### /test/route/2
| Metric | Actual |
| --- | --- |
| Hits | 16 |
| Error Rate | 0.00% |
| Mean Response Time | 84.69ms |
| Median Response Time | 8.50ms |
| 95th Percentile | 550.00ms |

`,
			`## Loadtest Results
### Score: 369.00
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
#### /test/route/1
| Metric | Actual |
| --- | --- |
| Hits | 25 |
| Error Rate | 40.00% |
| Mean Response Time | 23.67ms |
| Median Response Time | 8.00ms |
| 95th Percentile | 90.00ms |
| 90th Percentile | 70.00ms |
| Max Response Time | 100ms |
| Min Response Time | 1ms |
| Inter Quartile Range | 36 |

#### /test/route/2
| Metric | Actual |
| --- | --- |
| Hits | 16 |
| Error Rate | 0.00% |
| Mean Response Time | 84.69ms |
| Median Response Time | 8.50ms |
| 95th Percentile | 550.00ms |
| 90th Percentile | 90.00ms |
| Max Response Time | 1000ms |
| Min Response Time | 1ms |
| Inter Quartile Range | 45.5 |

`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			test := func(t *testing.T, verbose bool, expectedOutput string) {
				output := &strings.Builder{}
				config := &ltparse.ResultsConfig{
					Input:         strings.NewReader(testCase.Input),
					BaselineInput: nil,
					Output:        output,
					Display:       "markdown",
					Aggregate:     false,
					Verbose:       verbose,
				}

				err := ltparse.ParseResults(config)
				if testCase.ExpectedError {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, expectedOutput, output.String())
			}

			t.Run("normal", func(t *testing.T) {
				test(t, false, testCase.ExpectedOutput)
			})

			t.Run("verbose", func(t *testing.T) {
				test(t, true, testCase.ExpectedVerboseOutput)
			})
		})
	}
}
