package tokenizer

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

var (
	//go:embed js/gpt3-tokenizer.cjs.development.js
	tokenizerJs string

	//go:embed js/array-keyed-map.js
	arrayKeyedMapJs string

	//go:embed js/text.min.js
	fastTextEncodingJs string

	registry *require.Registry

	// optimize the alloc and instancing performance of
	// *goja.Runtime
	pool sync.Pool = sync.Pool{
		New: func() any {
			return newGojaRuntime()
		},
	}
)

type gojaRuntime struct {
	vm *goja.Runtime

	encode goja.Callable

	decode goja.Callable

	err error
}

type EncodeResult struct {
	Bpe  []int    `json:"bpe"`
	Text []string `json:"text"`
}

func init() {
	registry = require.NewRegistry(require.WithLoader(func(p string) ([]byte, error) {
		switch path.Base(p) {
		case "array-keyed-map":
			return []byte(arrayKeyedMapJs), nil
		case "fast-text-encoding":
			return []byte(fastTextEncodingJs), nil
		}
		return nil, require.IllegalModuleNameError
	}))

	runtime := pool.Get().(*gojaRuntime)
	if runtime.err != nil {
		panic(runtime.err)
	}

	pool.Put(runtime)
}

func newGojaRuntime() *gojaRuntime {
	vm := goja.New()
	registry.Enable(vm)
	_, err := vm.RunString(tokenizerJs + "\n" +
		`const tokenizer = new GPT3NodeTokenizer({type: 'gpt3'});
		 function encode(str) {return tokenizer.encode(str)}
		 function decode(tokens) {return tokenizer.decode(tokens)}`)
	if err != nil {
		return &gojaRuntime{
			vm:  vm,
			err: err,
		}
	}

	encode, decode, err := getEncodeAndDecodeFunctionsWithinGojaRuntime(vm)
	return &gojaRuntime{
		vm:     vm,
		encode: encode,
		decode: decode,
		err:    err,
	}
}

func getEncodeAndDecodeFunctionsWithinGojaRuntime(vm *goja.Runtime) (goja.Callable, goja.Callable, error) {
	encode, ok := goja.AssertFunction(vm.Get("encode"))
	if !ok {
		return nil, nil, errors.New("encode is not a function")
	}
	decode, ok := goja.AssertFunction(vm.Get("decode"))
	if !ok {
		return nil, nil, errors.New("decode is not a function")
	}

	return encode, decode, nil
}

// 计算token数
func CalcTokenCountV1(text string) int {
	r, err := EncodeV1(text)
	if err != nil {
		fmt.Printf("计算token错误：%v", err)
		return 0
	}

	return len(r.Bpe)
}

func MustCalTokenV1(str string) int {
	token, err := CalTokenV1(str)
	if err != nil {
		panic(err)
	}

	return token
}

func CalTokenV1(str string) (int, error) {
	r, err := EncodeV1(str)
	if err != nil {
		return 0, err
	}

	return len(r.Bpe), nil
}

func MustEncodeV1(str string) EncodeResult {
	r, err := EncodeV1(str)
	if err != nil {
		panic(err)
	}

	return *r
}

func EncodeV1(str string) (*EncodeResult, error) {
	gojaRuntime := pool.Get().(*gojaRuntime)
	if gojaRuntime.err != nil {
		return nil, gojaRuntime.err
	}
	defer pool.Put(gojaRuntime)

	v, err := gojaRuntime.encode(goja.Undefined(), gojaRuntime.vm.ToValue(str))
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(v.Export())
	r := &EncodeResult{}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, err
	}

	return r, nil
}

func MustDecodeV1(tokens []int) string {
	r, err := DecodeV1(tokens)
	if err != nil {
		panic(err)
	}

	return r
}

func DecodeV1(tokens []int) (string, error) {
	gojaRuntime := pool.Get().(*gojaRuntime)
	if gojaRuntime.err != nil {
		return "", gojaRuntime.err
	}
	defer pool.Put(gojaRuntime)

	v, err := gojaRuntime.decode(goja.Undefined(), gojaRuntime.vm.ToValue(tokens))
	if err != nil {
		return "", err
	}

	return v.String(), nil
}
