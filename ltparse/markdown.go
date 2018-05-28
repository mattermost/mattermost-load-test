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

	timingSummaryMarkdown = template.Must(template.New("timingSummaryMarkdown").Parse(
		`## Loadtest Results
### Score: {{printf "%.2f" .GetScore}}
The score is the the average of the mean reponse times below.

### Routes
`,
	))

	singleTimingTemplate = template.Must(template.New("singleTimingTemplate").Funcs(funcMap).Parse(
		`#### {{.Name}}
| Metric | Actual |
| --- | --- |
| Hits | {{.NumHits}} |
| Error Rate | {{printf "%.2f%%" .ErrorRate}} |
| Max Response Time | {{.Max}}ms |
| Min Response Time | {{.Min}}ms |
| Mean Response Time | {{printf "%.2f" .Mean}}ms |
| Median Response Time | {{printf "%.2f" .Median}}ms |
| Inter Quartile Range | {{.InterQuartileRange}} |
`,
	))

	comparisonTimingTemplate = template.Must(template.New("comparisonTimingTemplate").Funcs(funcMap).Parse(
		`#### {{.Actual.Name}}
| Metric | Baseline | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- |
| Hits | {{.Baseline.NumHits}} | {{.Actual.NumHits}} | {{compareInt64 .Actual.NumHits .Baseline.NumHits}} | {{comparePercentageInt64 .Actual.NumHits .Baseline.NumHits}}
| Error Rate | {{printf "%.2f%%" .Baseline.ErrorRate }} | {{printf "%.2f%%" .Actual.ErrorRate}} | {{comparePercentageFloat64 .Actual.ErrorRate .Baseline.ErrorRate}} | {{comparePercentageFloat64 .Actual.ErrorRate .Baseline.ErrorRate}} |
| Max Response Time | {{.Baseline.Max}}ms | {{.Actual.Max}}ms | {{compareFloat64 .Actual.Max .Baseline.Max}}ms | {{comparePercentageFloat64 .Actual.Max .Baseline.Max}} |
| Min Response Time | {{.Baseline.Min}}ms | {{.Actual.Min}}ms | {{compareFloat64 .Actual.Min .Baseline.Min}}ms | {{comparePercentageFloat64 .Actual.Min .Baseline.Min}} |
| Mean Response Time | {{printf "%.2f" .Baseline.Mean}}ms | {{printf "%.2f" .Actual.Mean}}ms | {{compareFloat64 .Actual.Mean .Baseline.Mean}}ms | {{comparePercentageFloat64 .Actual.Mean .Baseline.Mean}} |
| Median Response Time | {{printf "%.2f" .Baseline.Median}}ms | {{printf "%.2f" .Actual.Median}}ms | {{compareFloat64 .Actual.Median .Baseline.Median}}ms | {{comparePercentageFloat64 .Actual.Median .Baseline.Median}} |
| Inter Quartile Range | {{.Baseline.InterQuartileRange}} | {{.Actual.InterQuartileRange}} | {{compareFloat64 .Actual.InterQuartileRange .Baseline.InterQuartileRange}}ms | {{comparePercentageFloat64 .Actual.InterQuartileRange .Baseline.InterQuartileRange}} |
`,
	))

	comparisonTimingWithoutBaselineTemplate = template.Must(template.New("comparisonTimingWithoutBaselineTemplate").Funcs(funcMap).Parse(
		`#### {{.Name}}
| Metric | Baseline | Actual | Delta |
| --- | --- | --- | --- |
| Hits | - | {{.NumHits}} | - |
| Error Rate | - | {{printf "%.2f%%" .ErrorRate}} | - |
| Max Response Time | - | {{.Max}}ms | - |
| Min Response Time | - | {{.Min}}ms | - |
| Mean Response Time | - | {{printf "%.2f" .Mean}}ms | - |
| Median Response Time | - | {{printf "%.2f" .Median}}ms | - |
| Inter Quartile Range | - | {{.InterQuartileRange}} | - |
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

func dumpSingleTimingsMarkdown(timings *loadtest.ClientTimingStats, output io.Writer) error {
	if err := timingSummaryMarkdown.Execute(output, timings); err != nil {
		return errors.Wrap(err, "error executing summary template")
	}

	for _, route := range sortedRoutes(timings.Routes) {
		if err := singleTimingTemplate.Execute(output, route); err != nil {
			return errors.Wrap(err, "error executing route template")
		}
	}

	return nil
}

func dumpComparisonTimingsMarkdown(timings *loadtest.ClientTimingStats, baseline *loadtest.ClientTimingStats, output io.Writer) error {
	if err := timingSummaryMarkdown.Execute(output, timings); err != nil {
		return errors.Wrap(err, "error executing summary template")
	}

	for _, route := range sortedRoutes(timings.Routes) {
		baselineRoute, ok := baseline.Routes[route.Name]
		if !ok {
			if err := comparisonTimingWithoutBaselineTemplate.Execute(output, route); err != nil {
				return errors.Wrap(err, "error executing route template")
			}
		} else {
			comparison := struct {
				Actual   *loadtest.RouteStats
				Baseline *loadtest.RouteStats
			}{
				route,
				baselineRoute,
			}
			if err := comparisonTimingTemplate.Execute(output, comparison); err != nil {
				return errors.Wrap(err, "error executing route template")
			}
		}
	}

	return nil
}

func dumpTimingsMarkdown(timings *loadtest.ClientTimingStats, baselineTimings *loadtest.ClientTimingStats, output io.Writer) error {
	if baselineTimings == nil {
		return dumpSingleTimingsMarkdown(timings, output)
	} else {
		return dumpComparisonTimingsMarkdown(timings, baselineTimings, output)
	}
}
