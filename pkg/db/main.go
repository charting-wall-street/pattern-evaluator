package db

import (
	"fmt"
	"github.com/godoji/algocore/pkg/kiosk"
	"github.com/northberg/candlestick"
	"log"
	"sync"
)

var cacheLock = sync.Mutex{}
var cache = make(map[string][]*candlestick.CandleSet)

func GetCandles(interval int64, resolution int64, symbol string) []*candlestick.CandleSet {
	cacheKey := fmt.Sprintf("%d_%d_%s", interval, resolution, symbol)
	cacheLock.Lock()
	defer cacheLock.Unlock()
	if v, ok := cache[cacheKey]; ok {
		return v
	}
	candles, err := kiosk.GetAllCandles(interval, resolution, symbol)
	if err != nil {
		panic(err)
	}
	cache[cacheKey] = candles
	return candles
}

func CandleAtTimestamp(ts int64, collection []*candlestick.CandleSet) *candlestick.Candle {
	for _, set := range collection {
		if set == nil {
			log.Fatalln("candle set cannot be nil")
		}
		if ts < set.UnixFirst() || set.UnixLast() < ts {
			continue
		}
		c := set.AtTime(ts)
		if c.Missing {
			return nil
		}
		return c
	}
	return nil
}
