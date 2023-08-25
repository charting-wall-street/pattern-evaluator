package main

import (
	"encoding/gob"
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"log"
	"os"
	"path"
	"pattern-evaluator/pkg/bucket"
	"pattern-evaluator/pkg/config"
	"pattern-evaluator/pkg/evaluate"
	"pattern-evaluator/pkg/techniques"
	"pattern-evaluator/pkg/triplebarrier"
	"strings"
	"sync"
	"time"
)

func loadScenarios(algoName string, symbol string) []*algo.ScenarioSet {
	fileName := algoName + "_" + strings.ReplaceAll(symbol, ":", "_") + ".gob"
	outputPath := path.Join(".", "output", "events", fileName)

	f, err := os.Open(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	results := make([]*algo.ScenarioSet, 0)
	err = gob.NewDecoder(f).Decode(&results)
	if err != nil {
		panic(err)
	}
	return results
}

func findScenario(params evaluate.ParamSet, scenarios []*algo.ScenarioSet) *algo.ScenarioSet {
	for _, scenario := range scenarios {
		if scenario.Parameters[0] == params.Params[0] {
			return scenario
		}
	}
	return nil
}

func GatherForSymbol(algoName string, evaluator string, symbols []string) {

	fileName := evaluator + "_" + algoName + ".gob"
	outputPath := path.Join(".", "output", "metrics", fileName)

	if evv := techniques.GetHandler(evaluator); evv == nil {
		fmt.Printf("skipped: %s\n", fileName)
		return
	}

	combos, err := config.LoadCombinations("./params.txt")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	startTime := time.Now().UTC().UnixMilli()

	outputLock := sync.Mutex{}
	output := make(map[string][]evaluate.ResultItem, 0)
	for _, symbol := range symbols {
		results := make([]evaluate.ResultItem, 0)
		resultLock := sync.Mutex{}
		scenarios := loadScenarios(algoName, symbol)
		for _, combination := range combos {
			scenario := findScenario(combination, scenarios)
			wg.Add(1)
			semaphore <- struct{}{}
			go func(combo evaluate.ParamSet, sym string, events []*algo.Event) {
				defer wg.Done()
				ev := techniques.GetHandler(evaluator)
				metrics := ev.Evaluate(&combo, sym, events)
				conf := evaluate.EvalConfig{
					Name:    algoName,
					Symbol:  sym,
					Options: combo,
				}
				resultLock.Lock()
				results = append(results, evaluate.ResultItem{
					Config: conf,
					Result: metrics,
				})
				resultLock.Unlock()
				outputLock.Lock()
				output[symbol] = results
				outputLock.Unlock()
				<-semaphore
			}(combination, symbol, scenario.Events)
		}
	}
	wg.Wait()

	//checkMap := make(map[int64]int)
	//for s, items := range output {
	//	for _, item := range items {
	//		checkMap[item.Config.Options.Timeout] += item.Result.Size()
	//	}
	//	fmt.Println(algoName, evaluator, s, len(items), checkMap)
	//}

	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	gob.Register(triplebarrier.BarrierMetrics{})
	gob.Register(bucket.BucketMetrics{})
	err = gob.NewEncoder(f).Encode(output)
	if err != nil {
		panic(err)
	}

	elapsed := time.Now().UTC().UnixMilli() - startTime
	fmt.Printf("[%s, %s] Took %d milliseconds\n", algoName, evaluator, elapsed)
}

func main() {

	err := os.MkdirAll("./output/metrics", 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalln(err)
	}

	symbols, err := config.GetSymbolList()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for _, algoName := range config.GetAlgoList() {
		for _, ev := range techniques.GetTechniques() {
			wg.Add(1)
			go func(a string, e string, xs []string) {
				defer wg.Done()
				GatherForSymbol(a, e, xs)
			}(algoName, ev, symbols)
		}
	}
	wg.Wait()
}
