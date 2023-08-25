package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"pattern-evaluator/pkg/evaluate"
	"pattern-evaluator/pkg/triplebarrier"
	"strings"
)

func AggregateYearOverYear(inputPath string) {

	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	fileName := filepath.Base(inputPath)
	fileExt := filepath.Ext(fileName)
	fileNameNoExt := fileName[0 : len(fileName)-len(fileExt)]
	outputDir := filepath.Join(".", "output", "yoy")

	var metricsBySymbol map[string][]*evaluate.ResultItem
	err = gob.NewDecoder(f).Decode(&metricsBySymbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var m evaluate.Metrics
	for _, results := range metricsBySymbol {
		for _, result := range results {
			opts := result.Config.Options
			if !(opts.Threshold >= 0.02 && opts.Threshold <= 0.05) {
				continue
			}
			if opts.Timeout != 14 {
				continue
			}
			if m == nil {
				m = result.Result
			} else {
				m = m.Combine(result.Result)
			}
		}
	}

	o, err := os.Create(filepath.Join(outputDir, "yoy_"+strings.Split(fileNameNoExt, "_")[1]+".csv"))
	if err != nil {
		panic(err)
	}
	defer o.Close()
	o.WriteString("year,performance,ups,downs,limits\n")

	bm := m.(triplebarrier.BarrierMetrics)
	for y, ev := range bm.EventsByYear {
		ups := ev[triplebarrier.UpperHit]
		downs := ev[triplebarrier.LowerHit]
		limits := ev[triplebarrier.TimeLimit]
		line := fmt.Sprintf("%d,%f,%d,%d,%d\n", y, evaluate.Performance(ups, downs), ups, downs, limits)
		_, _ = o.WriteString(line)
	}
}

func main() {

	gob.Register(triplebarrier.BarrierMetrics{})

	err := os.MkdirAll(filepath.Join(".", "output", "yoy"), 0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	files, err := filepath.Glob("./output/metrics/*.gob")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, file := range files {
		if strings.Index(file, "barrier") == -1 {
			continue
		}
		fmt.Println(file)
		AggregateYearOverYear(file)
	}
}
