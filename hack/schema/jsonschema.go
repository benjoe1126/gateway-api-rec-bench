package main

import (
	"encoding/json"
	"fmt"
	"onlab-bm/pkg/scenario"

	"github.com/invopop/jsonschema"
)

func main() {
	sc := scenario.Scenario{}
	r := new(jsonschema.Reflector)
	schema := r.Reflect(sc)
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
