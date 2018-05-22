package ltparse

type RouteStats struct {
	NumHits            int64
	NumErrors          int64
	Duration           []float64
	Max                float64
	Min                float64
	Mean               float64
	Median             float64
	InterQuartileRange float64
}

type Timings struct {
	Routes map[string]*RouteStats
}

func (ts *Timings) GetScore() float64 {
	total := 0.0
	num := 0.0
	for _, stats := range ts.Routes {
		total += stats.Mean
		total += stats.Median
		total += stats.InterQuartileRange
		num += 1.0
	}

	return total / num
}
