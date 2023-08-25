package main

import (
	"encoding/gob"
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"github.com/northberg/candlestick"
	"log"
	"math"
	"os"
	"path"
	"pattern-evaluator/pkg/bucket"
	"pattern-evaluator/pkg/config"
	"pattern-evaluator/pkg/triplebarrier"
	"pattern-evaluator/pkg/validator"
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

func CalculateSD(values []float64, mul float64) (float64, float64) {
	// Step 1: Calculate the mean
	sum := 0.0
	for _, val := range values {
		sum += val * mul
	}
	mean := sum / float64(len(values))

	// Step 2: Calculate the sum of squared differences from the mean
	sumSquaredDiff := 0.0
	for _, val := range values {
		diff := (val * mul) - mean
		squaredDiff := math.Pow(diff, 2)
		sumSquaredDiff += squaredDiff
	}

	// Step 3: Calculate the variance
	variance := sumSquaredDiff / float64(len(values)-1)

	// Step 4: Calculate the standard deviation
	sd := math.Sqrt(variance)

	return mean, sd
}

func validateForLimit(symbols []string) {
	fmt.Print("Size delta between time limits: ")
	for _, symbol := range symbols {
		scenarios := loadScenarios("random", symbol)
		r1Sum := 0
		r2Sum := 0
		for _, scenario := range scenarios {
			events := scenario.Events
			r1 := triplebarrier.Evaluate(symbol, candlestick.Interval1d, events, 0.05, 3)
			r2 := triplebarrier.Evaluate(symbol, candlestick.Interval1d, events, 0.05, 7)
			r1Sum += r1.Size() + r1.Timeouts()
			r2Sum += r2.Size() + r2.Timeouts()
		}
		if r1Sum != r2Sum {
			log.Fatalln("size mismatch", r1Sum, r2Sum)
		}
	}
	fmt.Println("ok")
}

func validateForSymbol(symbols []string) {

	startTime := time.Now().UTC().UnixMilli()

	barrierElapsedDays := int64(0)
	barrierAverage := 0.0
	barrierResults := make([]float64, 0)
	bucketAverage := 0.0
	pointAverage := 0.0
	totalResults := 0
	for _, symbol := range symbols {
		scenarios := loadScenarios("random", symbol)
		for _, scenario := range scenarios {
			events := scenario.Events
			buckets := bucket.Evaluate(symbol, candlestick.Interval1d, events, 0.07, 500)
			barriers := triplebarrier.Evaluate(symbol, candlestick.Interval1d, events, 0.07, 500)
			returnAtPoint := validator.Evaluate(symbol, candlestick.Interval1d, events)

			if math.IsNaN(barriers.SumReturn) {
				fmt.Println("oh no")
			}

			// Point in time evaluator
			pointAverage += returnAtPoint

			// Triple barrier evaluator
			if barriers.Size() > 0 {
				r := barriers.SumReturn / float64(barriers.Size())
				barrierAverage += r
				barrierResults = append(barrierResults, r)
				barrierElapsedDays += barriers.SumTime / int64(barriers.Size())
			}

			// Bucket evaluator
			if buckets.Size() > 0 {
				bucketAverage += buckets.SumReturn / float64(buckets.Size())
			}

			totalResults++
		}
	}

	pointMetric := pointAverage / float64(totalResults) * 100
	barrierMetric := barrierAverage / float64(totalResults) * 100
	bucketMetric := bucketAverage / float64(totalResults) * 100

	fmt.Printf("Point:   %.5f%%", pointMetric)
	printConclusion(pointMetric)
	fmt.Printf("Barrier: %.2f%%", barrierMetric)
	printConclusion(barrierMetric)
	fmt.Printf("Bucket:  %.2f%%", bucketMetric)
	printConclusion(bucketMetric)

	daysPerTrade := barrierElapsedDays / int64(totalResults)
	ratio := 365 / float64(daysPerTrade)
	mean, sd := CalculateSD(barrierResults, ratio)

	fmt.Printf("Average days per trade: %d\n", daysPerTrade)
	fmt.Printf("Yearly expected return: mean=%.3f sd=%.3f n=%d\n", mean*100, sd*100, len(barrierResults))

	elapsed := time.Now().UTC().UnixMilli() - startTime
	fmt.Printf("Took %d milliseconds\n", elapsed)
}

func printConclusion(metric float64) {
	if metric < 1 {
		fmt.Println(" ok")
	} else {
		fmt.Println(" bad")
	}
}

func main() {
	fmt.Println("Evaluating performance of random trades")
	symbols, err := config.GetSymbolList()
	if err != nil {
		panic(err)
	}
	validateForSymbol(symbols)
	validateForLimit(symbols)
}
