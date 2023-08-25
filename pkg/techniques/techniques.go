package techniques

import (
	"pattern-evaluator/pkg/bucket"
	"pattern-evaluator/pkg/evaluate"
	"pattern-evaluator/pkg/triplebarrier"
)

var handlerMapping = map[string]evaluate.Evaluator{
	"barriers": &triplebarrier.Evaluator{},
	"buckets":  &bucket.Evaluator{},
}

func GetHandler(name string) evaluate.Evaluator {
	if e, ok := handlerMapping[name]; !ok {
		return nil
	} else {
		return e
	}
}

func GetTechniques() []string {
	xs := make([]string, 0)
	for x := range handlerMapping {
		xs = append(xs, x)
	}
	return xs
}
