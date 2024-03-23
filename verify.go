package grace

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	invalidParamCountFmt = "%s has %d output(s), but %s expects %d input(s)"
	invalidParamTypeFmt  = "between %s and %s: %s is not assignable to %s"
)

var NotFuncErr = errors.New("Fn must be a function")
var IncompatibleFunctionSignatureErr = errors.New("incompatible task function signatures")

func verifyTaskCompatibility(output, input *Task) error {
	if len(output.ReturnValueTypes) != input.Fn.Type().NumIn() {
		err := fmt.Errorf(invalidParamCountFmt, output.Name, len(output.ReturnValueTypes), input.Name, input.Fn.Type().NumIn())
		return errors.Join(IncompatibleFunctionSignatureErr, err)
	}

	for i, typ := range output.ReturnValueTypes {
		if !input.Fn.Type().In(i).AssignableTo(typ) {
			err := fmt.Errorf(invalidParamTypeFmt, output.Name, input.Name, output.ReturnValueTypes[i], input.Fn.Type().In(i))
			return errors.Join(IncompatibleFunctionSignatureErr, err)
		}
	}

	return nil
}

func verifyReturnType[T any](last *Task) error {
	typ := last.Fn.Type()
	if typ.NumOut() == 0 {
		return nil
	}

	retTyp := reflect.TypeFor[T]()
	outTyp := typ.Out(typ.NumOut() - 1)

	if !outTyp.AssignableTo(retTyp) {
		return fmt.Errorf("return type %s is not compatible to the last output type: %s", outTyp, retTyp)
	}

	return nil
}

func verifyInitialParams(firstFn reflect.Type, params []interface{}) error {
	paramSize := len(params)
	expectedSize := firstFn.NumIn()
	if paramSize != expectedSize {
		return fmt.Errorf("invalid input params: expected %v params but got %v", expectedSize, paramSize)
	}

	for i, p := range params {
		expectedType := firstFn.In(i)
		actualType := reflect.TypeOf(p)
		if !actualType.AssignableTo(expectedType) {
			return fmt.Errorf("invalid input params: expected %s at %d but got %s", expectedType, i+1, actualType)
		}
	}

	return nil
}

func isFunc(fn reflect.Type) error {
	if fn.Kind() != reflect.Func {
		return NotFuncErr
	}
	return nil
}
