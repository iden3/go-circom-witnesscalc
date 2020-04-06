package witnesscalc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	wasm3 "github.com/iden3/go-wasm3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestParams struct {
	wasmFilename   string
	inputsFilename string
	prime          string
	nVars          int32
	r              string
	rInv           string
	witness        string
}

func TestWitnessCalcMyCircuit1(t *testing.T) {
	testWitnessCalc(t, TestParams{
		wasmFilename:   "test_files/mycircuit.wasm",
		inputsFilename: "test_files/mycircuit-input1.json",
		prime:          "21888242871839275222246405745257275088548364400416034343698204186575808495617",
		nVars:          4,
		r:              "115792089237316195423570985008687907853269984665640564039457584007913129639936",
		rInv:           "9915499612839321149637521777990102151350674507940716049588462388200839649614",
		witness:        `["1","33","3","11"]`,
	}, true)
}

func TestWitnessCalcMyCircuit2(t *testing.T) {
	testWitnessCalc(t, TestParams{
		wasmFilename:   "test_files/mycircuit.wasm",
		inputsFilename: "test_files/mycircuit-input2.json",
		prime:          "21888242871839275222246405745257275088548364400416034343698204186575808495617",
		nVars:          4,
		r:              "115792089237316195423570985008687907853269984665640564039457584007913129639936",
		rInv:           "9915499612839321149637521777990102151350674507940716049588462388200839649614",
		witness:        `["1","21888242871839275222246405745257275088548364400416034343698204186575672693159","21888242871839275222246405745257275088548364400416034343698204186575796149939","11"]`,
	}, true)
}

func TestWitnessCalcMyCircuit3(t *testing.T) {
	testWitnessCalc(t, TestParams{
		wasmFilename:   "test_files/mycircuit.wasm",
		inputsFilename: "test_files/mycircuit-input3.json",
		prime:          "21888242871839275222246405745257275088548364400416034343698204186575808495617",
		nVars:          4,
		r:              "115792089237316195423570985008687907853269984665640564039457584007913129639936",
		rInv:           "9915499612839321149637521777990102151350674507940716049588462388200839649614",
		witness:        `["1","21888242871839275222246405745257275088548364400416034343698204186575808493616","10944121435919637611123202872628637544274182200208017171849102093287904246808","2"]`,
	}, true)
}

func TestWitnessCalcSmtVerifier10(t *testing.T) {
	witnessJSON, err := ioutil.ReadFile("test_files/smtverifier10-witness.json")
	if err != nil {
		panic(err)
	}
	testWitnessCalc(t, TestParams{
		wasmFilename:   "test_files/smtverifier10.wasm",
		inputsFilename: "test_files/smtverifier10-input.json",
		prime:          "21888242871839275222246405745257275088548364400416034343698204186575808495617",
		nVars:          4794,
		r:              "115792089237316195423570985008687907853269984665640564039457584007913129639936",
		rInv:           "9915499612839321149637521777990102151350674507940716049588462388200839649614",
		witness:        string(witnessJSON),
	}, false)
}

var testNConstraints = false

func TestWitnessCalcNConstraints(t *testing.T) {
	if !testNConstraints {
		return
	}
	oldWd, err := os.Getwd()
	require.Nil(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.Nil(t, err)
	}()
	err = os.Chdir(path.Join(oldWd, "test_files"))
	require.Nil(t, err)

	for i := 1; i < 8; i++ {
		// for i := 1; i < 3; i++ {
		n := int(math.Pow10(i))
		log.Printf("WitnessCalc with %v constraints\n", n)
		err := exec.Command("cp", "nconstraints.circom", "nconstraints.circom.tmp").Run()
		require.Nil(t, err)
		err = exec.Command("sed", "-i", fmt.Sprintf("s/{{N}}/%v/g", n), "nconstraints.circom.tmp").Run()
		require.Nil(t, err)
		err = exec.Command("./node_modules/.bin/circom", "nconstraints.circom.tmp", "-w", fmt.Sprintf("nconstraints-%v.wasm", n)).Run()
		if err != nil {
			fmt.Println(err)
		}
		require.Nil(t, err)

		wasmFilename := fmt.Sprintf("nconstraints-%v.wasm", n)
		var inputs = map[string]interface{}{"in": new(big.Int).SetInt64(2)}

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
		witnessCalculator, err := NewWitnessCalculator(runtime, module)
		require.Nil(t, err)
		p := witnessCalculator.prime
		start := time.Now()
		w, err := witnessCalculator.CalculateWitness(inputs, false)
		elapsed := time.Since(start)
		require.Nil(t, err)
		log.Printf("Took %v\n", elapsed)

		runtime.Destroy()

		out := new(big.Int).SetInt64(2)
		for i := 1; i < n; i++ {
			out.Mul(out, out)
			out.Add(out, new(big.Int).SetInt64(int64(i)))
			out.Mod(out, p)
		}

		assert.Equal(t, out, w[1])

		err = os.Remove("nconstraints.circom.tmp")
		require.Nil(t, err)
		err = os.Remove(fmt.Sprintf("nconstraints-%v.wasm", n))
		require.Nil(t, err)
	}
}

func testWitnessCalc(t *testing.T, p TestParams, logWitness bool) {
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
	require.Nil(t, err)
	log.Printf("Read WASM module (%d bytes)\n", len(wasmBytes))

	module, err := runtime.ParseModule(wasmBytes)
	require.Nil(t, err)
	module, err = runtime.LoadModule(module)
	require.Nil(t, err)
	log.Print("Loaded module")

	// fmt.Printf("NumImports: %v\n", module.NumImports())
	// fns, err := NewWitnessCalcFns(runtime, module)
	// if err != nil {
	// 	panic(err)
	// }

	inputsBytes, err := ioutil.ReadFile(p.inputsFilename)
	require.Nil(t, err)
	inputs, err := ParseInputs(inputsBytes)
	require.Nil(t, err)
	log.Print("Inputs: ", inputs)

	witnessCalculator, err := NewWitnessCalculator(runtime, module)
	require.Nil(t, err)
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
	require.Nil(t, err)
	if logWitness {
		log.Print("Witness: ", w)
	}
	wJSON, err := json.Marshal(WitnessJSON(w))
	require.Nil(t, err)
	if logWitness {
		log.Print("Witness JSON: ", string(wJSON))
	}
	pWitness := strings.ReplaceAll(p.witness, " ", "")
	pWitness = strings.ReplaceAll(pWitness, "\n", "")
	assert.Equal(t, pWitness, string(wJSON))

	// DEBUG
	// {
	// 	elemsCalc := strings.Split(string(wJSON), ",")
	// 	elems := strings.Split(p.witness, ",")
	// 	if len(elemsCalc) != len(elems) {
	// 		panic(fmt.Errorf("Witness length differs: %v, %v", len(elemsCalc), len(elems)))
	// 	}
	// 	for i := 0; i < len(elems); i++ {
	// 		fmt.Printf("exp %v\ngot %v\n\n", elems[i], elemsCalc[i])
	// 	}
	// }
	wb, err := witnessCalculator.CalculateBinWitness(inputs, false)
	require.Nil(t, err)
	if logWitness {
		log.Print("WitnessBin: ", hex.EncodeToString(wb))
	}
}
