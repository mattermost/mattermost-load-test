package ltparse

import (
	"fmt"
	"html/template"
	"os"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

func dumpTimingsText(timings *loadtest.ClientTimingStats) error {
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
		if err := rateTemplate.Execute(os.Stdout, route); err != nil {
			return errors.Wrap(err, "error executing template")
		}
	}

	total := 0.0
	num := 0.0
	for _, stats := range timings.Routes {
		total += stats.Mean
		total += stats.Median
		total += stats.InterQuartileRange
		num += 1.0
	}

	score := total / num

	fmt.Printf("Score: %.2f", score)
	fmt.Println("")

	return nil
}
