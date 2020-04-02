package witnesscalc

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
	"testing"

	wasm3 "github.com/iden3/go-wasm3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlatSlice(t *testing.T) {
	one := new(big.Int).SetInt64(1)
	two := new(big.Int).SetInt64(2)
	three := new(big.Int).SetInt64(3)
	four := new(big.Int).SetInt64(4)

	a := one
	fa := flatSlice(a)
	assert.Equal(t, []*big.Int{one}, fa)

	b := []*big.Int{one, two}
	fb := flatSlice(b)
	assert.Equal(t, []*big.Int{one, two}, fb)

	c := []interface{}{one, []*big.Int{two, three}}
	fc := flatSlice(c)
	assert.Equal(t, []*big.Int{one, two, three}, fc)

	d := []interface{}{[]*big.Int{one, two}, []*big.Int{three, four}}
	fd := flatSlice(d)
	assert.Equal(t, []*big.Int{one, two, three, four}, fd)
}

func TestParseInputs(t *testing.T) {
	one := new(big.Int).SetInt64(1)
	two := new(big.Int).SetInt64(2)
	three := new(big.Int).SetInt64(3)
	four := new(big.Int).SetInt64(4)

	a, err := ParseInputs([]byte(`{"a": 1, "b": "2"}`))
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"a": one, "b": two}, a)

	b, err := ParseInputs([]byte(`{"a": 1, "b": [2, 3]}`))
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"a": one, "b": []interface{}{two, three}}, b)

	c, err := ParseInputs([]byte(`{"a": 1, "b": [[1, 2], [3, 4]]}`))
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"a": one, "b": []interface{}{[]interface{}{one, two}, []interface{}{three, four}}}, c)
}

type TestParams struct {
	wasmFilename   string
	inputsFilename string
	prime          string
	nVars          int32
	r              string
	rInv           string
	witness        string
}

func TestWitnessCalcCircuit2(t *testing.T) {
	testWitnessCalc(t, TestParams{
		wasmFilename:   "test_files/mycircuit2.wasm",
		inputsFilename: "test_files/mycircuit2-input.json",
		prime:          "21888242871839275222246405745257275088548364400416034343698204186575808495617",
		nVars:          4,
		r:              "115792089237316195423570985008687907853269984665640564039457584007913129639936",
		rInv:           "9915499612839321149637521777990102151350674507940716049588462388200839649614",
		witness:        `["1","33","3","11"]`,
	})
}

func testWitnessCalc(t *testing.T, p TestParams) {
	log.Print("Initializing WASM3")

	runtime := wasm3.NewRuntime(&wasm3.Config{
		Environment: wasm3.NewEnvironment(),
		StackSize:   64 * 1024,
	})
	log.Println("Runtime ok")
	// err := runtime.ResizeMemory(16)
	// if err != nil {
	// 	panic(err)
	// }

	// log.Println("Runtime Memory len: ", len(runtime.Memory()))

	wasmBytes, err := ioutil.ReadFile(p.wasmFilename)
	if err != nil {
		panic(err)
	}
	log.Printf("Read WASM module (%d bytes)\n", len(wasmBytes))

	module, err := runtime.ParseModule(wasmBytes)
	if err != nil {
		panic(err)
	}
	module, err = runtime.LoadModule(module)
	if err != nil {
		panic(err)
	}
	log.Print("Loaded module")

	// fmt.Printf("NumImports: %v\n", module.NumImports())
	// fns, err := NewWitnessCalcFns(runtime, module)
	// if err != nil {
	// 	panic(err)
	// }

	inputsBytes, err := ioutil.ReadFile(p.inputsFilename)
	if err != nil {
		panic(err)
	}
	inputs, err := ParseInputs(inputsBytes)
	if err != nil {
		panic(err)
	}
	log.Print("Inputs: ", inputs)

	witnessCalculator, err := NewWitnessCalculator(runtime, module)
	if err != nil {
		panic(err)
	}
	log.Print("n32: ", witnessCalculator.n32)
	log.Print("prime: ", witnessCalculator.prime)
	log.Print("mask32: ", witnessCalculator.mask32)
	log.Print("nVars: ", witnessCalculator.nVars)
	log.Print("n64: ", witnessCalculator.n64)
	log.Print("r: ", witnessCalculator.r)
	log.Print("rInv: ", witnessCalculator.rInv)

	assert.Equal(t, p.prime, witnessCalculator.prime.String())
	assert.Equal(t, p.r, witnessCalculator.r.String())
	assert.Equal(t, p.rInv, witnessCalculator.rInv.String())
	assert.Equal(t, p.nVars, witnessCalculator.nVars)

	w, err := witnessCalculator.CalculateWitness(inputs, false)
	if err != nil {
		panic(err)
	}
	log.Print("Witness: ", w)
	wJSON, err := json.Marshal(WitnessJSON(w))
	if err != nil {
		panic(err)
	}
	log.Print("Witness JSON: ", string(wJSON))
	assert.Equal(t, p.witness, string(wJSON))
	wb, err := witnessCalculator.CalculateBinWitness(inputs, false)
	if err != nil {
		panic(err)
	}
	log.Print("WitnessBin: ", hex.EncodeToString(wb))
}
