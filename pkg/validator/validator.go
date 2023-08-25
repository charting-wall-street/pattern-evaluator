package validator

import (
	"github.com/godoji/algocore/pkg/algo"
	"github.com/northberg/candlestick"
	"pattern-evaluator/pkg/db"
)

func findOutcome(event *algo.Event, collection []*candlestick.CandleSet) (float64, bool) {
	entryTime := event.Time

	entryCandle := db.CandleAtTimestamp(entryTime, collection)
	if entryCandle.Open == 0.0 || entryCandle.Missing {
		return 0, false
	}
	if entryCandle != nil {
		return (entryCandle.Close - entryCandle.Open) / entryCandle.Open, true
	}

	panic("candle not found, invalid event!")
}

func Evaluate(symbol string, interval int64, events []*algo.Event) float64 {

	collection := db.GetCandles(interval, candlestick.Interval1d, symbol)

	average := 0.0
	for _, event := range events {
		outcome, ok := findOutcome(event, collection)
		if ok {
			average += outcome
		}
	}

	if len(events) == 0 {
		return 0
	}
	result := average / float64(len(events))
	return result
}
