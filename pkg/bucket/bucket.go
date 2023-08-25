package bucket

import (
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"github.com/northberg/candlestick"
	"log"
	"pattern-evaluator/pkg/db"
	"pattern-evaluator/pkg/evaluate"
)

type BucketMetrics struct {
	Modified  bool
	Undefined int
	Buckets   map[int]int
	SumReturn float64
}

type Evaluator struct{}

func (e *Evaluator) Evaluate(params *evaluate.ParamSet, symbol string, events []*algo.Event) evaluate.Metrics {
	return Evaluate(symbol, candlestick.Interval1d, events, params.Threshold, params.Timeout)
}

func (qm BucketMetrics) Combine(other evaluate.Metrics) evaluate.Metrics {
	if qm.Modified {
		log.Fatalln("cannot modify after emit")
	}

	combined := BucketMetrics{
		Modified:  qm.Modified,
		Undefined: qm.Undefined,
		Buckets:   make(map[int]int),
		SumReturn: qm.SumReturn,
	}

	for i, v := range qm.Buckets {
		combined.Buckets[i] = v
	}

	if otherMetrics, ok := other.(*BucketMetrics); ok {
		for i := 0; i < 4; i++ {
			combined.Buckets[i] += otherMetrics.Buckets[i]
		}
	} else if otherMetrics, ok := other.(BucketMetrics); ok {
		for i := 0; i < 4; i++ {
			combined.Buckets[i] += otherMetrics.Buckets[i]
		}
	} else {
		panic("cannot not add other type than BucketMetrics")
	}

	return combined
}

func (qm BucketMetrics) Evaluator() string {
	return "Triple Barrier"
}

func (qm BucketMetrics) Size() int {
	total := 0
	for _, v := range qm.Buckets {
		total += v
	}
	return total
}

func (qm BucketMetrics) String() string {
	output := ""
	for i := 0; i < 4; i++ {
		if v, ok := qm.Buckets[i]; ok {
			output += fmt.Sprintf("%d ", v)
		} else {
			output += "0 "
		}
	}
	return output
}

func (qm BucketMetrics) Value() float64 {
	return evaluate.Performance(qm.GetBucket(3), qm.GetBucket(0))
}

func (qm BucketMetrics) GetBucket(i int) int {
	if v, ok := qm.Buckets[i]; ok {
		return v
	} else {
		return 0
	}
}

func (qm BucketMetrics) Emit(key string) float64 {
	switch key {
	case "worst":
		return evaluate.Performance(qm.GetBucket(2)+qm.GetBucket(3), qm.GetBucket(0)+qm.GetBucket(1)) * 100
	case "balanced":
		return evaluate.Performance(qm.GetBucket(3), qm.GetBucket(0)) * 100
	case "size":
		return float64(qm.Size())
	case "wins":
		return float64(qm.GetBucket(2) + qm.GetBucket(3))
	default:
		panic("undefined emit key")
	}
}

func findOutcome(event *algo.Event, threshold float64, timeout int64, interval int64, collection []*candlestick.CandleSet) (float64, bool) {

	// First point in time, where we have knowledge of the event
	bookTime := event.Time + interval

	// Find the candle at the start time, or after the start time if no candle was available
	var startCandle *candlestick.Candle
	for i := int64(0); i < 10; i++ {
		entryCandle := db.CandleAtTimestamp(bookTime+i*interval, collection)
		if entryCandle != nil {
			startCandle = entryCandle
			break
		}
	}
	if startCandle == nil || startCandle.Open == 0.0 {
		return 0, false
	}

	// The entry price of our trade would be at the opening of the start candle
	entryPrice := startCandle.Open

	// The time at which we entered the trade
	startTime := startCandle.Time

	// Determine the upper and lower barrier
	upperBarrier := entryPrice * (1.0 + threshold)
	lowerBarrier := entryPrice * (1.0 - threshold)

	// Make sure that we don't hit the time limit because of a missing candle, we still accept an exit at the first available
	missing := 0
	lastCandle := startCandle

	for i := int64(0); i < timeout; i++ {
		currentCandle := db.CandleAtTimestamp(startTime+i*interval, collection)

		// Check if the current candle is missing, otherwise skip evaluating it
		if currentCandle == nil {
			missing++
			continue
		} else {
			missing = 0
		}

		lastCandle = currentCandle
		low := currentCandle.Low
		high := currentCandle.High

		if low <= lowerBarrier {
			profit := (lowerBarrier - startCandle.Open) / startCandle.Open
			return profit, true
		}
		if high >= upperBarrier {
			profit := (upperBarrier - startCandle.Open) / startCandle.Open
			return profit, true
		}
	}

	// Fall back on the last seen candle
	return (lastCandle.Close - startCandle.Open) / startCandle.Open, true
}

func Evaluate(symbol string, interval int64, events []*algo.Event, threshold float64, timeout int64) *BucketMetrics {

	collection := db.GetCandles(interval, candlestick.Interval1d, symbol)

	m := &BucketMetrics{
		Modified:  false,
		Undefined: 0,
		Buckets:   make(map[int]int, 4),
		SumReturn: 0,
	}

	for _, event := range events {
		exit, ok := findOutcome(event, threshold, timeout, interval, collection)

		if ok {
			m.SumReturn += exit
			if exit > 0 && exit < threshold/2 {
				m.Buckets[2]++
			} else if exit < 0 && exit > -threshold/2 {
				m.Buckets[1]++
			} else if exit > 0 {
				m.Buckets[3]++
			} else if exit < 0 {
				m.Buckets[0]++
			}
		} else {
			m.Undefined++
		}
	}

	return m
}
