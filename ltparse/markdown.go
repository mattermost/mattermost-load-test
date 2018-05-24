package ltparse

import (
	"fmt"
	"html/template"
	"os"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-load-test/loadtest"
)

func dumpTimingsMarkdown(timings *loadtest.ClientTimingStats) error {
	const rates = `##### {{.Name}}
| Metric | Score |
| --- | --- |
| Hits | {{.NumHits}} |
| Error Rate | {{percent .NumErrors .NumHits}}% |
| Max Response Time | {{.Max}}ms |
| Min Response Time | {{.Min}}ms |
| Mean Response Time | {{printf "%.2f" .Mean}}ms |
| Median Response Time | {{printf "%.2f" .Median}}ms |
| Inter Quartile Range | {{.InterQuartileRange}} |
`
	funcMap := template.FuncMap{
		"percent": func(x, y int64) string {
			return fmt.Sprintf("%.2f", float64(x)/float64(y)*100.0)
		},
	}
	rateTemplate := template.Must(template.New("rates").Funcs(funcMap).Parse(rates))

	fmt.Println("### Loadtest Results")
	fmt.Printf("#### Score: %.2f\n", timings.GetScore())
	fmt.Println("This score is the the average mean of the routes below.")

	fmt.Println("#### Routes")
	for _, route := range timings.Routes {
		if err := rateTemplate.Execute(os.Stdout, route); err != nil {
			return errors.Wrap(err, "error executing template")
		}
	}

	return nil
}
