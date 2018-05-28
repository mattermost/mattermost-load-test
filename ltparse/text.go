package ltparse

import (
	"fmt"
	"html/template"
	"io"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

func dumpTimingsText(timings *loadtest.ClientTimingStats, output io.Writer) error {
	const rates = `Total Hits: {{.NumHits}}
Error Rate: {{percent .NumErrors .NumHits}}%
Max Response Time: {{.Max}}ms
Min Response Time: {{.Min}}ms
Mean Response Time: {{printf "%.2f" .Mean}}ms
Median Response Time: {{printf "%.2f" .Median}}ms
Inter Quartile Range: {{.InterQuartileRange}}

`
	funcMap := template.FuncMap{
		"percent": func(x, y int64) string {
			return fmt.Sprintf("%.2f", float64(x)/float64(y)*100.0)
		},
	}
	rateTemplate := template.Must(template.New("rates").Funcs(funcMap).Parse(rates))

	fmt.Println("--------- Timings Report ------------")

	for routeName, route := range timings.Routes {
		fmt.Println("Route: " + routeName)
		if err := rateTemplate.Execute(output, route); err != nil {
			return errors.Wrap(err, "error executing template")
		}
	}

	fmt.Printf("Score: %.2f", timings.GetScore())
	fmt.Println("")

	return nil
}
