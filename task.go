package grace

import (
	"errors"
	"fmt"
	"reflect"
)

// Task is a reflected task that will run.
// Name: the Name of this task
// Fn: the internal function that will run.
// ReturnValueTypes: types of the return values. IMPORTANT: this will not include the error return
type Task struct {
	Name             string
	Fn               reflect.Value
	ReturnValueTypes []reflect.Type
	hasErrorOut      bool
	Cleanup          func() error
}

var (
	emptyTypes = make([]reflect.Type, 0)
)

type TaskConfig struct {
	Name    string
	Cleanup func() error
	Fn      interface{}
}

// NewTask creates new Task instance
func NewTask(config *TaskConfig) (*Task, error) {
	if config.Fn == nil {
		return nil, fmt.Errorf("failed to create task: '%s': Fn cannot be nil", config.Name)
	}

	fnType := reflect.TypeOf(config.Fn)

	var err error

	if err = isFunc(fnType); err != nil {
		return nil, fmt.Errorf("failed to create task '%s': %w", config.Name, err)
	}

	hasErrOut := hasErrorOut(fnType)

	return &Task{
		Fn:               reflect.ValueOf(config.Fn),
		Name:             config.Name,
		Cleanup:          config.Cleanup,
		hasErrorOut:      hasErrOut,
		ReturnValueTypes: getReturnValueTypes(fnType, hasErrOut),
	}, nil
}

// Run executes a function with given params
// the result of the Run will exclude the last error output if its Fn has an error out.
func (t *Task) Run(params []reflect.Value) ([]reflect.Value, error) {
	ret := t.Fn.Call(params)

	if t.hasErrorOut { // has error out
		if err, ok := ret[len(ret)-1].Interface().(error); ok && err != nil {
			return t.doCleanup(nil, err)
		} else {
			return t.doCleanup(ret[:len(ret)-1], nil)
		}
	}

	// return results as-is
	return t.doCleanup(ret, nil)
}

func (t *Task) doCleanup(v []reflect.Value, err error) ([]reflect.Value, error) {
	if t.Cleanup != nil {
		if cuErr := t.Cleanup(); cuErr != nil {
			return v, errors.Join(cuErr, err)
		}
	}
	return v, err
}

func hasErrorOut(fn reflect.Type) bool {
	return fn.NumOut() > 0 && fn.Out(fn.NumOut()-1).AssignableTo(reflect.TypeFor[error]())
}

// getReturnValueTypes returns values-only types that excludes the last error type if exists
func getReturnValueTypes(fn reflect.Type, hasErrorOut bool) []reflect.Type {
	if fn.NumOut() == 0 {
		return emptyTypes
	}

	// check and decrease the loop size if Fn returns an error
	size := fn.NumOut()
	if hasErrorOut {
		size--
	}

	types := make([]reflect.Type, size)
	for i := 0; i < size; i++ {
		types[i] = fn.Out(i)
	}

	return types
}
