package config

import (
	"bufio"
	"github.com/godoji/algocore/pkg/kiosk"
	"os"
	"pattern-evaluator/pkg/evaluate"
	"strconv"
	"strings"
)

var algoList = []string{"double-top", "double-bottom", "random", "triple-top", "triple-bottom", "head-and-shoulders"}
var evalList = []string{"3b", "fixed", "bucket"}

type EvalParams struct {
	Thresholds   []float64
	TimeLimits   []int64
	HighLowRange []float64
	HighLowTest  []float64
}

func GetAlgoList() []string {
	return algoList
}

func GetSymbolList() ([]string, error) {
	info, err := kiosk.GetExchangeInfo()
	if err != nil {
		return nil, err
	}
	symbols := make([]string, 0)
	for _, exchange := range info.Exchanges {
		if exchange.ExchangeId == "US" {
			for symbol := range exchange.Symbols {
				if symbol == "UNICORN:US:ZNH" {
					continue
				}
				symbols = append(symbols, symbol)
			}
		}
	}
	return symbols, nil
}

func LoadCombinations(filename string) ([]evaluate.ParamSet, error) {
	params, err := LoadEvaluationParameters(filename)
	if err != nil {
		return nil, err
	}
	var combinations []evaluate.ParamSet
	for _, threshold := range params.Thresholds {
		for _, timeout := range params.TimeLimits {
			for _, p1 := range params.HighLowRange {
				combinations = append(combinations, evaluate.ParamSet{Threshold: threshold, Timeout: timeout, Params: []float64{p1}})
			}
		}
	}
	return combinations, nil
}

func LoadEvaluationParameters(filename string) (*EvalParams, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var thresholds []float64
	var timeLimits []int64
	var highLowRange []float64
	var highLowTest []float64

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		switch lineNumber {
		case 0:
			for _, field := range fields {
				val, err := strconv.ParseFloat(field, 64)
				if err != nil {
					return nil, err
				}
				thresholds = append(thresholds, val)
			}
		case 1:
			for _, field := range fields {
				val, err := strconv.ParseInt(field, 10, 64)
				if err != nil {
					return nil, err
				}
				timeLimits = append(timeLimits, val)
			}
		case 2:
			for _, field := range fields {
				val, err := strconv.ParseFloat(field, 64)
				if err != nil {
					return nil, err
				}
				highLowRange = append(highLowRange, val)
			}
		case 3:
			for _, field := range fields {
				val, err := strconv.ParseFloat(field, 64)
				if err != nil {
					return nil, err
				}
				highLowTest = append(highLowTest, val)
			}
		}
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &EvalParams{
		Thresholds:   thresholds,
		TimeLimits:   timeLimits,
		HighLowRange: highLowRange,
		HighLowTest:  highLowTest,
	}, nil
}
