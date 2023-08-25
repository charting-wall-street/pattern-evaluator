package triplebarrier

import (
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"github.com/northberg/candlestick"
	"math"
	"pattern-evaluator/pkg/db"
	"pattern-evaluator/pkg/evaluate"
	"time"
)

type BarrierEvent int

const (
	UpperHit BarrierEvent = iota
	LowerHit
	TimeLimit
	Undefined
)

type BarrierMetrics struct {
	Events       map[BarrierEvent]int
	EventsByYear map[int]map[BarrierEvent]int
	SumReturn    float64
	SumTime      int64
}

type Evaluator struct {
}

func (e *Evaluator) Evaluate(params *evaluate.ParamSet, symbol string, events []*algo.Event) evaluate.Metrics {
	return Evaluate(symbol, candlestick.Interval1d, events, params.Threshold, params.Timeout)
}

func (bm BarrierMetrics) Combine(other evaluate.Metrics) evaluate.Metrics {
	combined := BarrierMetrics{
		Events:       make(map[BarrierEvent]int),
		EventsByYear: make(map[int]map[BarrierEvent]int),
	}

	for event, count := range bm.Events {
		combined.Events[event] = count
	}
	for year, metrics := range bm.EventsByYear {
		combined.EventsByYear[year] = make(map[BarrierEvent]int)
		for e, i := range metrics {
			combined.EventsByYear[year][e] = i
		}
	}

	// now add the other BarrierMetrics
	if otherMetrics, ok := other.(BarrierMetrics); ok {
		for event, count := range otherMetrics.Events {
			combined.Events[event] += count
		}
		for year, metrics := range otherMetrics.EventsByYear {
			if _, ok := combined.EventsByYear[year]; !ok {
				combined.EventsByYear[year] = make(map[BarrierEvent]int)
			}
			for e, i := range metrics {
				combined.EventsByYear[year][e] += i
			}
		}
	} else {
		panic("cannot not add other type than BarrierMetrics")
	}

	return combined
}

func (bm BarrierMetrics) Evaluator() string {
	return "Triple Barrier"
}

func (bm BarrierMetrics) Size() int {
	return bm.Events[UpperHit] + bm.Events[LowerHit]
}

func (bm BarrierMetrics) UpTrends() int {
	return bm.Events[UpperHit]
}

func (bm BarrierMetrics) DownTrends() int {
	return bm.Events[LowerHit]
}

func (bm BarrierMetrics) Timeouts() int {
	return bm.Events[TimeLimit]
}

func (bm BarrierMetrics) Value() float64 {
	return evaluate.Performance(bm.UpTrends(), bm.DownTrends())
}

func (bm BarrierMetrics) String() string {
	return fmt.Sprintf("%d/%d+%d", bm.UpTrends(), bm.DownTrends(), bm.Timeouts())
}

func (bm BarrierMetrics) Emit(key string) float64 {
	switch key {
	case "worst":
		return evaluate.Performance(bm.UpTrends(), bm.DownTrends()+bm.Timeouts()) * 100
	case "balanced":
		return evaluate.Performance(bm.UpTrends(), bm.DownTrends()) * 100
	case "size":
		return float64(bm.Size())
	case "wins":
		return float64(bm.UpTrends())
	default:
		panic("undefined emit key")
	}
}

func findOutcome(event *algo.Event, threshold float64, timeLimit int64, interval int64, collection []*candlestick.CandleSet) (BarrierEvent, float64, int64) {

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
		return Undefined, 0, 0
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

	// Keep iterating candles till we either hit a barrier, or reach the time limit in candles, starting from the opening candle
	for i := int64(0); i < timeLimit || (missing > 1 && missing < 5); i++ {
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
			return LowerHit, profit, i
		}
		if high >= upperBarrier {
			profit := (upperBarrier - startCandle.Open) / startCandle.Open
			return UpperHit, profit, i
		}
	}

	profit := (lastCandle.Close - startCandle.Open) / startCandle.Open
	return TimeLimit, profit, timeLimit
}

func Evaluate(symbol string, interval int64, events []*algo.Event, threshold float64, timeout int64) *BarrierMetrics {

	// Retrieve a list of all candles for a given symbol, adjusted for splits
	collection := db.GetCandles(interval, candlestick.Interval1d, symbol)

	// Create the mapping in which we will store our barrier hits
	m := &BarrierMetrics{
		Events:       make(map[BarrierEvent]int),
		EventsByYear: make(map[int]map[BarrierEvent]int),
		SumReturn:    0,
		SumTime:      0,
	}

	// Iterate all events after which we expect effect
	for _, event := range events {
		year := time.Unix(event.Time, 0).UTC().Year()
		result, profit, elapsed := findOutcome(event, threshold, timeout, interval, collection)
		if math.IsNaN(profit) {
			panic("profit cannot be nan")
		}
		m.Events[result]++
		m.SumReturn += profit
		m.SumTime += elapsed
		if _, ok := m.EventsByYear[year]; !ok {
			m.EventsByYear[year] = make(map[BarrierEvent]int)
		}
		m.EventsByYear[year][result]++
	}

	return m
}
