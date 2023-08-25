package evaluate

import (
	"encoding/csv"
	"fmt"
	"github.com/godoji/algocore/pkg/algo"
	"os"
)

type EvalConfig struct {
	Name    string   `json:"name"`
	Symbol  string   `json:"symbol"`
	Options ParamSet `json:"options"`
}

type ParamSet struct {
	Threshold float64   `json:"threshold"`
	Timeout   int64     `json:"timeout"`
	Params    []float64 `json:"params"`
}

type ResultItem struct {
	Config EvalConfig
	Result Metrics
}

type ResultItemYoY struct {
	Config EvalConfig
	Result map[int]Metrics
}

type MetricsTable struct {
	Columns []string
	Rows    []string
	Values  MetricsGrid
}

type MetricsGrid = [][]Metrics

func CombineMatrix(dst *MetricsGrid, other MetricsGrid) {
	for i := range *dst {
		for j := range (*dst)[i] {
			if (*dst)[i][j] == nil {
				(*dst)[i][j] = other[i][j]
			} else if other[i][j] != nil {
				(*dst)[i][j] = (*dst)[i][j].Combine(other[i][j])
			}
		}
	}
}

func DumpMetrics(g MetricsGrid, key string, filePath string, rowNames []string, colNames []string) {

	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	cols := len(colNames)

	writer := csv.NewWriter(file)
	headerRow := make([]string, cols+1)
	for i, f := range colNames {
		headerRow[i+1] = fmt.Sprintf("%s", f)
	}
	err = writer.Write(headerRow)
	if err != nil {
		panic(err)
	}
	for i, row := range g {
		stringRow := make([]string, cols+1)
		stringRow[0] = fmt.Sprintf("%s", rowNames[i])
		for j, val := range row {
			if val == nil {
				stringRow[j+1] = "nil"
			} else if key == "string" {
				stringRow[j+1] = val.String()
			} else if key == "size" {
				stringRow[j+1] = fmt.Sprintf("%d", val.Size())
			} else if key == "wins" {
				stringRow[j+1] = fmt.Sprintf("%.0f", val.Emit(key))
			} else {
				stringRow[j+1] = fmt.Sprintf("%.2f", val.Emit(key))
			}
		}
		err := writer.Write(stringRow)
		if err != nil {
			panic(err)
		}
	}

	writer.Flush()
}

func Performance(wins int, losses int) float64 {
	totalTrades := wins + losses
	if totalTrades == 0 {
		return 0.0
	}
	winRate := float64(wins) / float64(totalTrades)
	return winRate
}

type DiffMetrics struct {
	Base Metrics
	Data Metrics
}

func (d DiffMetrics) Evaluator() string {
	return d.Data.Evaluator() + "-" + d.Base.Evaluator()
}

func (d DiffMetrics) String() string {
	panic("string not available for diff metrics")
}

func (d DiffMetrics) Value() float64 {
	return d.Data.Value() - d.Base.Value()
}

func (d DiffMetrics) Size() int {
	panic("returning size of diff metrics makes no sense")
}

func (d DiffMetrics) Combine(other Metrics) Metrics {
	panic("combining off diff metrics is not possible")
}

func (d DiffMetrics) Emit(key string) float64 {
	return d.Data.Emit(key) - d.Base.Emit(key)
}

func DiffMetricsTables(src *MetricsTable, base *MetricsTable) *MetricsTable {
	arr := make([][]Metrics, len(src.Values))
	for i, value := range src.Values {
		arr[i] = make([]Metrics, len(value))
		for j, metrics := range value {
			arr[i][j] = DiffMetrics{
				Base: base.Values[i][j],
				Data: metrics,
			}
		}
	}
	return &MetricsTable{
		Columns: src.Columns,
		Rows:    src.Rows,
		Values:  arr,
	}
}

type Metrics interface {
	Evaluator() string
	String() string
	Value() float64
	Size() int
	Combine(other Metrics) Metrics
	Emit(key string) float64
}

type Evaluator interface {
	Evaluate(params *ParamSet, symbol string, events []*algo.Event) Metrics
}
