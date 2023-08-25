package main

import (
	"encoding/gob"
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"os"
	"path"
	"pattern-evaluator/pkg/config"
	"strings"
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

func GatherForSymbol(algoName string, symbols []string) {

	params, err := config.LoadEvaluationParameters("./params.txt")
	if err != nil {
		panic(err)
	}

	startTime := time.Now().UTC().UnixMilli()

	counter := make(map[float64]int)
	for _, symbol := range symbols {
		scenarios := loadScenarios(algoName, symbol)
		for _, f := range params.HighLowTest {
			for _, scenario := range scenarios {
				if scenario.Parameters[0] == f {
					counter[f] += len(scenario.Events)
				}
			}

		}

	}

	elapsed := time.Now().UTC().UnixMilli() - startTime
	fmt.Printf("[%s] Took %d milliseconds\n", algoName, elapsed)

	entries := make([]string, 0)
	for f, i := range counter {
		entries = append(entries, fmt.Sprintf("%0.f: %d", f, i))
	}
	fmt.Println("{", strings.Join(entries, ", "), "}")

}

func main() {

	symbols, err := config.GetSymbolList()
	if err != nil {
		panic(err)
	}

	for _, algoName := range config.GetAlgoList() {
		if algoName == "random" {
			continue
		}
		GatherForSymbol(algoName, symbols)
	}
}
