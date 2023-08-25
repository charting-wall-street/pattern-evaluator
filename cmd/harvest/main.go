package main

import (
	"encoding/gob"
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"github.com/godoji/algocore/pkg/kiosk"
	"github.com/northberg/candlestick"
	"os"
	"path"
	"pattern-evaluator/pkg/config"
	"strings"
	"sync"
	"time"
)

func HarvestEvents(algoName string, symbol string, params []float64) *algo.ScenarioSet {
	if algoName == "random" {
		res, err := kiosk.GetAlgorithm("random", candlestick.Interval1d, symbol, []float64{0.01}, true)
		if err != nil {
			panic(err)
		}
		res.Parameters[0] = params[0]
		return res
	} else {
		res, err := kiosk.GetAlgorithm(algoName, candlestick.Interval1d, symbol, params, true)
		if err != nil {
			panic(err)
		}
		return res
	}
}

func HarvestForSymbol(algoName string, symbol string) {

	fileName := algoName + "_" + strings.ReplaceAll(symbol, ":", "_") + ".gob"
	outputPath := path.Join(".", "output", "events", fileName)

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		return
	}

	hyperParams, err := config.LoadEvaluationParameters("./params.txt")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 15)
	startTime := time.Now().UTC().UnixMilli()
	resultLock := sync.Mutex{}
	results := make([]*algo.ScenarioSet, 0)

	for _, highLowParam := range hyperParams.HighLowTest {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(p float64) {
			defer wg.Done()
			events := HarvestEvents(algoName, symbol, []float64{p})
			resultLock.Lock()
			results = append(results, events)
			resultLock.Unlock()
			<-semaphore
		}(highLowParam)
	}
	wg.Wait()

	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = gob.NewEncoder(f).Encode(results)
	if err != nil {
		panic(err)
	}

	elapsed := time.Now().UTC().UnixMilli() - startTime
	if elapsed < 20 {
		fmt.Printf("[%s -> %s] Took %d milliseconds\n", algoName, symbol, elapsed)
	}
}

func main() {

	err := os.MkdirAll("./output/events", 0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	symbols, err := config.GetSymbolList()
	if err != nil {
		panic(err)
	}

	for _, symbol := range symbols {
		var wg sync.WaitGroup
		for _, algoName := range config.GetAlgoList() {
			wg.Add(1)
			go func(a string, s string) {
				defer wg.Done()
				HarvestForSymbol(a, s)
			}(algoName, symbol)
		}
		wg.Wait()
	}
}
