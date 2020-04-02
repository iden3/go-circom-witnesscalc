# go-witnesscalc

[![GoDoc](https://godoc.org/github.com/iden3/go-witnesscalc?status.svg)](https://godoc.org/github.com/iden3/go-witnesscalc)

Witness Calculator in go, calling WASM

## Example

```go
package main

import (
	"encoding/json"
	"io/ioutil"

	wasm3 "github.com/iden3/go-wasm3"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	wasmFilename :=   "test_files/mycircuit.wasm"
	inputsFilename := "test_files/mycircuit-input2.json"

	runtime := wasm3.NewRuntime(&wasm3.Config{
		Environment: wasm3.NewEnvironment(),
		StackSize:   64 * 1024,
	})

	wasmBytes, err := ioutil.ReadFile(wasmFilename)
	require.Nil(t, err)

	module, err := runtime.ParseModule(wasmBytes)
	require.Nil(t, err)

	module, err = runtime.LoadModule(module)
	require.Nil(t, err)

	inputsBytes, err := ioutil.ReadFile(inputsFilename)
	require.Nil(t, err)

	inputs, err := ParseInputs(inputsBytes)
	require.Nil(t, err)

	witnessCalculator, err := NewWitnessCalculator(runtime, module)
	require.Nil(t, err)

	w, err := witnessCalculator.CalculateWitness(inputs, false)
	require.Nil(t, err)

	wJSON, err := json.Marshal(WitnessJSON(w))
	require.Nil(t, err)
        fmt.Print(string(wJSON))
}
```

# License

GPLv3
