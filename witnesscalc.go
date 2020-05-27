package witnesscalc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"unsafe"

	log "github.com/sirupsen/logrus"

	wasm3 "github.com/iden3/go-wasm3"
)

// witnessCalcFns are wrapper functions to the WitnessCalc WASM module
type witnessCalcFns struct {
	getFrLen          func() (int32, error)
	getPRawPrime      func() (int32, error)
	getNVars          func() (int32, error)
	init              func(sanityCheck int32) error
	getSignalOffset32 func(pR, component, hashMSB, hashLSB int32) error
	setSignal         func(cIdx, component, signal, pVal int32) error
	getPWitness       func(w int32) (int32, error)
	getWitnessBuffer  func() (int32, error)
}

func getStack(sp unsafe.Pointer, length int) []uint64 {
	var data = (*uint64)(sp)
	var header reflect.SliceHeader
	header = *(*reflect.SliceHeader)(unsafe.Pointer(&header))
	header.Data = uintptr(unsafe.Pointer(data))
	header.Len = int(length)
	header.Cap = int(length)
	return *(*[]uint64)(unsafe.Pointer(&header))
}

func getMem(r *wasm3.Runtime, _mem unsafe.Pointer) []byte {
	var data = (*uint8)(_mem)
	length := r.GetAllocatedMemoryLength()
	var header reflect.SliceHeader
	header = *(*reflect.SliceHeader)(unsafe.Pointer(&header))
	header.Data = uintptr(unsafe.Pointer(data))
	header.Len = int(length)
	header.Cap = int(length)
	return *(*[]byte)(unsafe.Pointer(&header))
}

func getStr(mem []byte, p uint64) string {
	var buf bytes.Buffer
	for ; mem[p] != 0; p++ {
		buf.WriteByte(mem[p])
	}
	return buf.String()
}

// newWitnessCalcFns builds the witnessCalcFns from the loaded WitnessCalc WASM
// module in the runtime.  Imported functions (logging) are binded to dummy functions.
func newWitnessCalcFns(r *wasm3.Runtime, m *wasm3.Module, wc *WitnessCalculator) (*witnessCalcFns, error) {
	r.AttachFunction("runtime", "error", "v(iiiiii)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, _mem unsafe.Pointer) int {
			// func(code, pstr, a, b, c, d)

			stack := getStack(sp, 6)
			mem := getMem(r, _mem)

			code := stack[0]
			pstr := stack[1]
			a := stack[2]
			b := stack[3]
			c := stack[4]
			d := stack[5]

			var errStr string
			if code == 7 {
				errStr = fmt.Sprintf("%s %v != %v %s",
					getStr(mem, pstr),
					wc.loadFr(int32(b)), wc.loadFr(int32(c)), getStr(mem, d))
			} else {
				errStr = fmt.Sprintf("%s %v %v %v %v",
					getStr(mem, pstr), a, b, c, getStr(mem, d))
			}
			log.Errorf("WitnessCalculator WASM Error (%v): %v", code, errStr)
			return 0
		},
	))
	r.AttachFunction("runtime", "logSetSignal", "v(ii)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, mem unsafe.Pointer) int {
			return 0
		},
	))
	r.AttachFunction("runtime", "logGetSignal", "v(ii)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, mem unsafe.Pointer) int {
			return 0
		},
	))
	r.AttachFunction("runtime", "logFinishComponent", "v(i)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, mem unsafe.Pointer) int {
			return 0
		},
	))
	r.AttachFunction("runtime", "logStartComponent", "v(i)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, mem unsafe.Pointer) int {
			return 0
		},
	))
	r.AttachFunction("runtime", "log", "v(i)", wasm3.CallbackFunction(
		func(runtime wasm3.RuntimeT, sp unsafe.Pointer, mem unsafe.Pointer) int {
			return 0
		},
	))

	_getFrLen, err := r.FindFunction("getFrLen")
	if err != nil {
		return nil, err
	}
	getFrLen := func() (int32, error) {
		res, err := _getFrLen()
		if err != nil {
			return 0, err
		}
		return res.(int32), nil
	}
	_getPRawPrime, err := r.FindFunction("getPRawPrime")
	if err != nil {
		return nil, err
	}
	getPRawPrime := func() (int32, error) {
		res, err := _getPRawPrime()
		if err != nil {
			return 0, err
		}
		return res.(int32), nil
	}
	_getNVars, err := r.FindFunction("getNVars")
	if err != nil {
		return nil, err
	}
	getNVars := func() (int32, error) {
		res, err := _getNVars()
		if err != nil {
			return 0, err
		}
		return res.(int32), nil
	}
	_init, err := r.FindFunction("init")
	if err != nil {
		return nil, err
	}
	init := func(sanityCheck int32) error {
		_, err := _init(sanityCheck)
		if err != nil {
			return err
		}
		return nil
	}
	_getSignalOffset32, err := r.FindFunction("getSignalOffset32")
	if err != nil {
		return nil, err
	}
	getSignalOffset32 := func(pR, component, hashMSB, hashLSB int32) error {
		_, err := _getSignalOffset32(pR, component, hashMSB, hashLSB)
		if err != nil {
			return err
		}
		return nil
	}
	_setSignal, err := r.FindFunction("setSignal")
	if err != nil {
		return nil, err
	}
	setSignal := func(cIdx, component, signal, pVal int32) error {
		_, err := _setSignal(cIdx, component, signal, pVal)
		if err != nil {
			return err
		}
		return nil
	}
	_getPWitness, err := r.FindFunction("getPWitness")
	if err != nil {
		return nil, err
	}
	getPWitness := func(w int32) (int32, error) {
		res, err := _getPWitness(w)
		if err != nil {
			return 0, err
		}
		return res.(int32), nil
	}
	_getWitnessBuffer, err := r.FindFunction("getWitnessBuffer")
	if err != nil {
		return nil, err
	}
	getWitnessBuffer := func() (int32, error) {
		res, err := _getWitnessBuffer()
		if err != nil {
			return 0, err
		}
		return res.(int32), nil
	}

	return &witnessCalcFns{
		getFrLen:          getFrLen,
		getPRawPrime:      getPRawPrime,
		getNVars:          getNVars,
		init:              init,
		getSignalOffset32: getSignalOffset32,
		setSignal:         setSignal,
		getPWitness:       getPWitness,
		getWitnessBuffer:  getWitnessBuffer,
	}, nil
}

// WitnessJSON is a wrapper type to Marshal the Witness in JSON format
type WitnessJSON []*big.Int

// MarshalJSON marshals the WitnessJSON where each value is encoded in base 10
// as a string in an array.
func (w WitnessJSON) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for i, bi := range w {
		buffer.WriteString(`"` + bi.String() + `"`)
		if i != len(w)-1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// loadBigInt loads a *big.Int from the runtime memory at position p.
func loadBigInt(runtime *wasm3.Runtime, p int32, n int32) *big.Int {
	bigIntBytes := make([]byte, n)
	copy(bigIntBytes, runtime.Memory()[p:p+n])
	return new(big.Int).SetBytes(swap(bigIntBytes))
}

// WitnessCalculator is the object that allows performing witness calculation
// from signal inputs using the WitnessCalc WASM module.
type WitnessCalculator struct {
	n32    int32
	prime  *big.Int
	mask32 *big.Int
	nVars  int32
	n64    uint
	r      *big.Int
	rInv   *big.Int

	shortMax *big.Int
	shortMin *big.Int

	runtime *wasm3.Runtime
	fns     *witnessCalcFns
}

// NewWitnessCalculator creates a new WitnessCalculator from the WitnessCalc
// loaded WASM module in the runtime.
func NewWitnessCalculator(runtime *wasm3.Runtime, module *wasm3.Module) (*WitnessCalculator, error) {
	var wc WitnessCalculator
	fns, err := newWitnessCalcFns(runtime, module, &wc)
	if err != nil {
		return nil, err
	}

	n32, err := fns.getFrLen()
	if err != nil {
		return nil, err
	}
	// n32 = (n32 >> 2) - 2
	n32 = n32 - 8

	pRawPrime, err := fns.getPRawPrime()
	if err != nil {
		return nil, err
	}

	prime := loadBigInt(runtime, pRawPrime, n32)

	mask32 := new(big.Int).SetUint64(0xFFFFFFFF)
	nVars, err := fns.getNVars()
	if err != nil {
		return nil, err
	}

	n64 := uint(((prime.BitLen() - 1) / 64) + 1)
	r := new(big.Int).SetInt64(1)
	r.Lsh(r, n64*64)
	rInv := new(big.Int).ModInverse(r, prime)

	shortMax, ok := new(big.Int).SetString("0x80000000", 0)
	if !ok {
		return nil, fmt.Errorf("unable to set shortMax from string")
	}
	shortMin := new(big.Int).Set(prime)
	shortMin.Sub(shortMin, shortMax)

	wc.n32 = n32
	wc.prime = prime
	wc.mask32 = mask32
	wc.nVars = nVars
	wc.n64 = n64
	wc.r = r
	wc.rInv = rInv
	wc.shortMin = shortMin
	wc.shortMax = shortMax
	wc.runtime = runtime
	wc.fns = fns
	return &wc, nil
}

// loadBigInt loads a *big.Int from the runtime memory at position p.
func (wc *WitnessCalculator) loadBigInt(p int32, n int32) *big.Int {
	return loadBigInt(wc.runtime, p, n)
}

var zero32 [32]byte

// storeBigInt stores a *big.Int into the runtime memory at position p.
func (wc *WitnessCalculator) storeBigInt(p int32, v *big.Int) {
	bigIntBytes := swap(v.Bytes())
	copy(wc.runtime.Memory()[p:p+32], zero32[:])
	copy(wc.runtime.Memory()[p:p+int32(len(bigIntBytes))], bigIntBytes)
}

// memFreePos gives the next free runtime memory position.
func (wc *WitnessCalculator) memFreePos() int32 {
	return int32(binary.LittleEndian.Uint32(wc.runtime.Memory()[:4]))
}

// setMemFreePos sets the next free runtime memory position.
func (wc *WitnessCalculator) setMemFreePos(p int32) {
	binary.LittleEndian.PutUint32(wc.runtime.Memory()[:4], uint32(p))
}

// allocInt reserves space in the runtime memory and returns its position.
func (wc *WitnessCalculator) allocInt() int32 {
	p := wc.memFreePos()
	wc.setMemFreePos(p + 8)
	return p
}

// allocFr reserves space in the runtime memory for a Field element and returns its position.
func (wc *WitnessCalculator) allocFr() int32 {
	p := wc.memFreePos()
	wc.setMemFreePos(p + wc.n32*4 + 8)
	return p
}

// getInt loads an int32 from the runtime memory at position p.
func (wc *WitnessCalculator) getInt(p int32) int32 {
	return int32(binary.LittleEndian.Uint32(wc.runtime.Memory()[p : p+4]))
}

// setInt stores an int32 in the runtime memory at position p.
func (wc *WitnessCalculator) setInt(p, v int32) {
	binary.LittleEndian.PutUint32(wc.runtime.Memory()[p:p+4], uint32(v))
}

// setShortPositive stores a small positive Field element in the runtime memory at position p.
func (wc *WitnessCalculator) setShortPositive(p int32, v *big.Int) {
	if !v.IsInt64() || v.Int64() >= 0x80000000 {
		panic(fmt.Errorf("v should be < 0x80000000"))
	}
	wc.setInt(p, int32(v.Int64()))
	wc.setInt(p+4, 0)
}

// setShortPositive stores a small negative *big.Int in the runtime memory at position p.
func (wc *WitnessCalculator) setShortNegative(p int32, v *big.Int) {
	vNeg := new(big.Int).Set(wc.prime) // prime
	vNeg.Sub(vNeg, wc.shortMax)        // prime - max
	vNeg.Sub(v, vNeg)                  // v - (prime - max)
	vNeg.Add(wc.shortMax, vNeg)        // max + (v - (prime - max))
	if !vNeg.IsInt64() || vNeg.Int64() < 0x80000000 || vNeg.Int64() >= 0x80000000*2 {
		panic(fmt.Errorf("v should be < 0x80000000"))
	}
	wc.setInt(p, int32(vNeg.Int64()))
	wc.setInt(p+4, 0)
}

// setShortPositive stores a normal Field element in the runtime memory at position p.
func (wc *WitnessCalculator) setLongNormal(p int32, v *big.Int) {
	wc.setInt(p, 0)
	wc.setInt(p+4, math.MinInt32) // math.MinInt32 = 0x80000000
	wc.storeBigInt(p+8, v)
}

// storeFr stores a Field element in the runtime memory at position p.
func (wc *WitnessCalculator) storeFr(p int32, v *big.Int) {
	if v.Cmp(wc.shortMax) == -1 {
		wc.setShortPositive(p, v)
	} else if v.Cmp(wc.shortMin) >= 0 {
		wc.setShortNegative(p, v)
	} else {
		wc.setLongNormal(p, v)
	}
}

// fromMontgomery transforms a Field element from Montgomery form to regular form.
func (wc *WitnessCalculator) fromMontgomery(v *big.Int) *big.Int {
	res := new(big.Int).Set(v)
	res.Mul(res, wc.rInv)
	res.Mod(res, wc.prime)
	return res
}

// loadFr loads a Field element from the runtime memory at position p.
func (wc *WitnessCalculator) loadFr(p int32) *big.Int {
	m := wc.runtime.Memory()
	if (m[p+4+3] & 0x80) != 0 {
		res := wc.loadBigInt(p+8, wc.n32)
		if (m[p+4+3] & 0x40) != 0 {
			return wc.fromMontgomery(res)
		} else {
			return res
		}
	} else {
		if (m[p+3] & 0x40) != 0 {
			res := wc.loadBigInt(p, 4) // res
			res.Sub(res, wc.shortMax)  // res - max
			res.Add(wc.prime, res)     // res - max + prime
			res.Sub(res, wc.shortMax)  // res - max + (prime - max)
			return res
		} else {
			return wc.loadBigInt(p, 4)
		}
	}
}

// doCalculateWitness is an internal function that calculates the witness.
func (wc *WitnessCalculator) doCalculateWitness(inputs map[string]interface{}, sanityCheck bool) error {
	sanityCheckVal := int32(0)
	if sanityCheck {
		sanityCheckVal = 1
	}
	if err := wc.fns.init(sanityCheckVal); err != nil {
		return err
	}
	pSigOffset := wc.allocInt()
	pFr := wc.allocFr()

	for inputName, inputValue := range inputs {
		hMSB, hLSB := fnvHash(inputName)
		wc.fns.getSignalOffset32(pSigOffset, 0, hMSB, hLSB)
		sigOffset := wc.getInt(pSigOffset)
		fSlice := flatSlice(inputValue)
		for i, value := range fSlice {
			wc.storeFr(pFr, value)
			wc.fns.setSignal(0, 0, sigOffset+int32(i), pFr)
		}
	}

	return nil
}

// CalculateWitness calculates the witness given the inputs.
func (wc *WitnessCalculator) CalculateWitness(inputs map[string]interface{}, sanityCheck bool) ([]*big.Int, error) {
	oldMemFreePos := wc.memFreePos()

	if err := wc.doCalculateWitness(inputs, sanityCheck); err != nil {
		return nil, err
	}

	w := make([]*big.Int, wc.nVars)
	for i := int32(0); i < wc.nVars; i++ {
		pWitness, err := wc.fns.getPWitness(i)
		if err != nil {
			return nil, err
		}
		w[i] = wc.loadFr(pWitness)
	}

	wc.setMemFreePos(oldMemFreePos)
	return w, nil
}

// CalculateWitness calculates the witness in binary given the inputs.
func (wc *WitnessCalculator) CalculateBinWitness(inputs map[string]interface{}, sanityCheck bool) ([]byte, error) {
	oldMemFreePos := wc.memFreePos()

	if err := wc.doCalculateWitness(inputs, sanityCheck); err != nil {
		return nil, err
	}
	pWitnessBuff, err := wc.fns.getWitnessBuffer()
	if err != nil {
		return nil, err
	}
	witnessBuff := make([]byte, uint(wc.nVars)*wc.n64*8)
	copy(witnessBuff, wc.runtime.Memory()[pWitnessBuff:int(pWitnessBuff)+len(witnessBuff)])

	wc.setMemFreePos(oldMemFreePos)
	return witnessBuff, nil
}
