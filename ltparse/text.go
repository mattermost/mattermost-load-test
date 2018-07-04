package ltparse

import (
	"fmt"
	"html/template"
	"io"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

const text = `Total Hits: {{.Actual.NumHits}}
Error Rate: {{percent .Actual.ErrorRate}}
Mean Response Time: {{printf "%.2f" .Actual.Mean}}ms
Median Response Time: {{printf "%.2f" .Actual.Median}}ms
95th Percentile: {{printf "%.2f" .Actual.Percentile95}}ms
{{if .Verbose -}}
90th Percentile: {{printf "%.2f" .Actual.Percentile90}}ms
Max Response Time: {{.Actual.Max}}ms
Min Response Time: {{.Actual.Min}}ms
Inter Quartile Range: {{.Actual.InterQuartileRange}}
{{end}}
`

func dumpTimingsText(timings *loadtest.ClientTimingStats, output io.Writer, verbose bool) error {
	funcMap := template.FuncMap{
		"percent": func(x float64) string {
			return fmt.Sprintf("%.2f%%", float64(x)*100.0)
		},
	}
	rateTemplate := template.Must(template.New("rates").Funcs(funcMap).Parse(text))

	fmt.Fprint(output, "--------- Timings Report ------------\n")

	for _, route := range sortedRoutes(timings.Routes) {
		fmt.Fprintf(output, "Route: %s\n", route.Name)
		data := templateData{route, nil, verbose}
		if err := rateTemplate.Execute(output, data); err != nil {
			return errors.Wrap(err, "error executing template")
		}
	}

	fmt.Fprintf(output, "Score: %.2f\n", timings.GetScore())

	return nil
}
