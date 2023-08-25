package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"pattern-evaluator/pkg/bucket"
	"pattern-evaluator/pkg/config"
	"pattern-evaluator/pkg/evaluate"
	"pattern-evaluator/pkg/triplebarrier"
)

const defaultTimeLimit = 14

func StatThresholdVsRange(results []*evaluate.ResultItem, thresholds []float64, highLowParams []float64) evaluate.MetricsGrid {
	rows := len(thresholds)
	cols := len(highLowParams)
	arr := make([][]evaluate.Metrics, rows)
	for i := range arr {
		arr[i] = make([]evaluate.Metrics, cols)
	}

	for i, threshold := range thresholds {
		for j, highLowParam := range highLowParams {
			for _, result := range results {
				if result.Config.Options.Params[0] != highLowParam {
					continue
				}
				if result.Config.Options.Threshold != threshold {
					continue
				}
				if result.Config.Options.Timeout != defaultTimeLimit {
					continue
				}
				if arr[i][j] == nil {
					arr[i][j] = result.Result
				} else {
					arr[i][j] = arr[i][j].Combine(result.Result)
				}
			}
		}
	}

	return arr
}

func StatThresholdVsLimit(results []*evaluate.ResultItem, thresholds []float64, timeouts []int64) evaluate.MetricsGrid {
	rows := len(thresholds)
	cols := len(timeouts)
	arr := make([][]evaluate.Metrics, rows)
	for i := range arr {
		arr[i] = make([]evaluate.Metrics, cols)
	}

	for i, threshold := range thresholds {
		for j, timeLimit := range timeouts {
			for _, result := range results {
				if result.Config.Options.Threshold != threshold {
					continue
				}
				if result.Config.Options.Timeout != timeLimit {
					continue
				}
				if arr[i][j] == nil {
					arr[i][j] = result.Result
				} else {
					arr[i][j] = arr[i][j].Combine(result.Result)
				}
			}

		}
	}

	return arr
}

func AggregateStats(inputPath string) {

	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	fileName := filepath.Base(inputPath)
	fileExt := filepath.Ext(fileName)
	fileNameNoExt := fileName[0 : len(fileName)-len(fileExt)]
	outputDir := filepath.Join(".", "output", "tables")

	var metricsBySymbol map[string][]*evaluate.ResultItem
	err = gob.NewDecoder(f).Decode(&metricsBySymbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//checkMap := make(map[int64]int)
	//for s, items := range metricsBySymbol {
	//	for _, item := range items {
	//		checkMap[item.Config.Options.Timeout] += item.Result.Size()
	//	}
	//	fmt.Println(fileName, s, len(items), checkMap)
	//}

	hp, err := config.LoadEvaluationParameters("./params.txt")
	if err != nil {
		panic(err)
	}

	thresholdNames := make([]string, 0)
	for _, v := range hp.Thresholds {
		thresholdNames = append(thresholdNames, fmt.Sprintf("thld:%.3f", v))
	}
	highLowNames := make([]string, 0)
	for _, v := range hp.HighLowRange {
		highLowNames = append(highLowNames, fmt.Sprintf("rng:%.2f", v))
	}
	limitNames := make([]string, 0)
	for _, v := range hp.TimeLimits {
		limitNames = append(limitNames, fmt.Sprintf("limit:%d", v))
	}

	var byRange *evaluate.MetricsGrid
	for _, items := range metricsBySymbol {
		grid := StatThresholdVsRange(items, hp.Thresholds, hp.HighLowRange)
		if byRange == nil {
			byRange = &grid
		} else {
			evaluate.CombineMatrix(byRange, grid)
		}
	}

	fRange, err := os.Create(filepath.Join(outputDir, "by-range_"+fileNameNoExt+".gob"))
	if err != nil {
		panic(err)
	}
	defer fRange.Close()

	err = gob.NewEncoder(fRange).Encode(&evaluate.MetricsTable{
		Columns: highLowNames,
		Rows:    thresholdNames,
		Values:  *byRange,
	})
	if err != nil {
		panic(err)
	}

	var byLimit *evaluate.MetricsGrid
	for _, items := range metricsBySymbol {
		grid := StatThresholdVsLimit(items, hp.Thresholds, hp.TimeLimits)
		if byLimit == nil {
			byLimit = &grid
		} else {
			evaluate.CombineMatrix(byLimit, grid)
		}
	}

	fLimit, err := os.Create(filepath.Join(outputDir, "by-limit_"+fileNameNoExt+".gob"))
	if err != nil {
		panic(err)
	}
	defer fLimit.Close()

	err = gob.NewEncoder(fLimit).Encode(&evaluate.MetricsTable{
		Columns: limitNames,
		Rows:    thresholdNames,
		Values:  *byLimit,
	})
	if err != nil {
		panic(err)
	}
}

func main() {

	gob.Register(triplebarrier.BarrierMetrics{})
	gob.Register(bucket.BucketMetrics{})

	err := os.MkdirAll(filepath.Join(".", "output", "tables"), 0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	files, err := filepath.Glob("./output/metrics/*.gob")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, file := range files {
		AggregateStats(file)
	}
}
