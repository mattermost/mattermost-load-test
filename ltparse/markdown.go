package ltparse

import (
	"fmt"
	"io"
	"sort"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

var (
	funcMap = template.FuncMap{
		"compareInt64": func(a, b int64) string {
			delta := a - b
			if delta == 0 {
				return "0"
			} else {
				return fmt.Sprintf("%+d", delta)
			}
		},
		"compareFloat64": func(a, b float64) string {
			delta := a - b
			if delta == 0 {
				return "0"
			} else {
				return fmt.Sprintf("%+0.2f", delta)
			}
		},
		"comparePercentageInt64": func(a, b int64) string {
			delta := a - b
			if delta == 0 {
				return "0%"
			} else if b > 0 {
				return fmt.Sprintf("%+0.2f%%", float64(delta)/float64(b)*100)
			} else {
				return "-"
			}
		},
		"comparePercentageFloat64": func(a, b float64) string {
			delta := a - b
			if delta == 0 {
				return "0%"
			} else if b > 0 {
				return fmt.Sprintf("%+0.2f%%", delta/b*100)
			} else {
				return "-"
			}
		},
	}

	singleTimingSummaryMarkdown = template.Must(template.New("singleTimingSummaryMarkdown").Funcs(funcMap).Parse(
		`## Loadtest Results
### Score: {{printf "%.2f" .Actual.GetScore}}
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
`,
	))

	singleTimingTemplate = template.Must(template.New("singleTimingTemplate").Funcs(funcMap).Parse(
		`#### {{.Actual.Name}}
| Metric | Actual |
| --- | --- |
| Hits | {{.Actual.NumHits}} |
| Error Rate | {{printf "%.2f%%" .Actual.ErrorRate}} |
| Mean Response Time | {{printf "%.2f" .Actual.Mean}}ms |
| Median Response Time | {{printf "%.2f" .Actual.Median}}ms |
| 95th Percentile | {{printf "%.2f" .Actual.Percentile95}}ms |
{{if .Verbose -}}
| 90th Percentile | {{printf "%.2f" .Actual.Percentile90}}ms |
| Max Response Time | {{.Actual.Max}}ms |
| Min Response Time | {{.Actual.Min}}ms |
| Inter Quartile Range | {{.Actual.InterQuartileRange}} |
{{end}}
`,
	))

	comparisonTimingSummaryMarkdown = template.Must(template.New("comparisonTimingSummaryMarkdown").Funcs(funcMap).Parse(
		`## Loadtest Results
### Score: {{printf "%.2f" .Actual.GetScore}} ({{compareFloat64 .Actual.GetScore .Baseline.GetScore}}, relative to baseline)
The score is the average of the 95th percentile, median and interquartile ranges in the routes below.

### Routes
`,
	))

	comparisonTimingTemplate = template.Must(template.New("comparisonTimingTemplate").Funcs(funcMap).Parse(
		`#### {{.Actual.Name}}
| Metric | Baseline | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- |
| Hits | {{.Baseline.NumHits}} | {{.Actual.NumHits}} | {{compareInt64 .Actual.NumHits .Baseline.NumHits}} | {{comparePercentageInt64 .Actual.NumHits .Baseline.NumHits}}
| Error Rate | {{printf "%.2f%%" .Baseline.ErrorRate }} | {{printf "%.2f%%" .Actual.ErrorRate}} | {{comparePercentageFloat64 .Actual.ErrorRate .Baseline.ErrorRate}} | {{comparePercentageFloat64 .Actual.ErrorRate .Baseline.ErrorRate}} |
| Mean Response Time | {{printf "%.2f" .Baseline.Mean}}ms | {{printf "%.2f" .Actual.Mean}}ms | {{compareFloat64 .Actual.Mean .Baseline.Mean}}ms | {{comparePercentageFloat64 .Actual.Mean .Baseline.Mean}} |
| Median Response Time | {{printf "%.2f" .Baseline.Median}}ms | {{printf "%.2f" .Actual.Median}}ms | {{compareFloat64 .Actual.Median .Baseline.Median}}ms | {{comparePercentageFloat64 .Actual.Median .Baseline.Median}} |
| 95th Percentile | {{printf "%.2f" .Baseline.Percentile95}}ms | {{printf "%.2f" .Actual.Percentile95}}ms | {{compareFloat64 .Actual.Percentile95 .Baseline.Percentile95}}ms | {{comparePercentageFloat64 .Actual.Percentile95 .Baseline.Percentile95}} |
{{if .Verbose -}}
| 90th Percentile | {{printf "%.2f" .Baseline.Percentile90}}ms | {{printf "%.2f" .Actual.Percentile90}}ms | {{compareFloat64 .Actual.Percentile90 .Baseline.Percentile90}}ms | {{comparePercentageFloat64 .Actual.Percentile90 .Baseline.Percentile90}} |
| Max Response Time | {{.Baseline.Max}}ms | {{.Actual.Max}}ms | {{compareFloat64 .Actual.Max .Baseline.Max}}ms | {{comparePercentageFloat64 .Actual.Max .Baseline.Max}} |
| Min Response Time | {{.Baseline.Min}}ms | {{.Actual.Min}}ms | {{compareFloat64 .Actual.Min .Baseline.Min}}ms | {{comparePercentageFloat64 .Actual.Min .Baseline.Min}} |
| Inter Quartile Range | {{.Baseline.InterQuartileRange}} | {{.Actual.InterQuartileRange}} | {{compareFloat64 .Actual.InterQuartileRange .Baseline.InterQuartileRange}}ms | {{comparePercentageFloat64 .Actual.InterQuartileRange .Baseline.InterQuartileRange}} |
{{end}}
`,
	))

	comparisonTimingWithoutBaselineTemplate = template.Must(template.New("comparisonTimingWithoutBaselineTemplate").Funcs(funcMap).Parse(
		`#### {{.Name}}
| Metric | Baseline | Actual | Delta |
| --- | --- | --- | --- |
| Hits | - | {{.Actual.NumHits}} | - |
| Error Rate | - | {{printf "%.2f%%" .Actual.ErrorRate}} | - |
| Mean Response Time | - | {{printf "%.2f" .Actual.Mean}}ms | - |
| Median Response Time | - | {{printf "%.2f" .Actual.Median}}ms | - |
| 95th Percentile | - | {{printf "%.2f" .Actual.Percentile95}}ms | - |
{{if .Verbose -}}
| 90th Percentile | - | {{printf "%.2f" .Actual.Percentile90}}ms | - |
| Max Response Time | - | {{.Actual.Max}}ms | - |
| Min Response Time | - | {{.Actual.Min}}ms | - |
| Inter Quartile Range | - | {{.Actual.InterQuartileRange}} | - |
{{end}}
`,
	))
)

func sortedRoutes(routesMap map[string]*loadtest.RouteStats) []*loadtest.RouteStats {
	routeNames := make([]string, 0, len(routesMap))
	for routeName := range routesMap {
		routeNames = append(routeNames, routeName)
	}
	sort.Strings(routeNames)

	routes := make([]*loadtest.RouteStats, 0, len(routesMap))
	for _, routeName := range routeNames {
		routes = append(routes, routesMap[routeName])
	}

	return routes
}

func dumpSingleTimingsMarkdown(timings *loadtest.ClientTimingStats, output io.Writer, verbose bool) error {
	summaryData := struct {
		Actual *loadtest.ClientTimingStats
	}{
		timings,
	}
	if err := singleTimingSummaryMarkdown.Execute(output, summaryData); err != nil {
		return errors.Wrap(err, "error executing summary template")
	}

	for _, route := range sortedRoutes(timings.Routes) {
		data := templateData{route, nil, verbose}

		if err := singleTimingTemplate.Execute(output, data); err != nil {
			return errors.Wrap(err, "error executing route template")
		}
	}

	return nil
}

func dumpComparisonTimingsMarkdown(timings *loadtest.ClientTimingStats, baseline *loadtest.ClientTimingStats, output io.Writer, verbose bool) error {
	summaryData := struct {
		Actual   *loadtest.ClientTimingStats
		Baseline *loadtest.ClientTimingStats
	}{
		timings,
		baseline,
	}
	if err := comparisonTimingSummaryMarkdown.Execute(output, summaryData); err != nil {
		return errors.Wrap(err, "error executing summary template")
	}

	for _, route := range sortedRoutes(timings.Routes) {
		if baselineRoute, ok := baseline.Routes[route.Name]; !ok {
			data := templateData{route, nil, verbose}
			if err := comparisonTimingWithoutBaselineTemplate.Execute(output, data); err != nil {
				return errors.Wrap(err, "error executing route template")
			}
		} else {
			data := templateData{route, baselineRoute, verbose}
			if err := comparisonTimingTemplate.Execute(output, data); err != nil {
				return errors.Wrap(err, "error executing route template")
			}
		}
	}

	return nil
}

func dumpTimingsMarkdown(timings *loadtest.ClientTimingStats, baselineTimings *loadtest.ClientTimingStats, output io.Writer, verbose bool) error {
	if baselineTimings == nil {
		return dumpSingleTimingsMarkdown(timings, output, verbose)
	} else {
		return dumpComparisonTimingsMarkdown(timings, baselineTimings, output, verbose)
	}
}
