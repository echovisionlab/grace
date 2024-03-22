package grace

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

const paramCountMismatchErrFmt = "failed to run task chain %s: parameter count does not match. expected: %v, actual: %v"

// TaskChain consists a series of *Task to be run as a sequence
type TaskChain[T any] struct {
	Tasks     []*Task // 실행할 태스크들
	Cleanup   func() error
	Name      string
	hasOutput bool
}

type TaskChainConfig struct {
	Name    string
	Cleanup func() error
	Tasks   []*Task
}

// NewTaskChain returns a new TaskChain instance
func NewTaskChain[T any](config *TaskChainConfig) (*TaskChain[T], error) {
	tasks := config.Tasks

	for i := 0; i < len(tasks); i++ {
		if tasks[i] == nil {
			tasks = append(tasks[:i], tasks[i+1:]...)
			i--
		}
	}

	size := len(tasks)

	if size == 0 {
		return &TaskChain[T]{
			Tasks:     make([]*Task, 0),
			hasOutput: false,
		}, nil
	}

	for i := 1; i < size; i++ {
		prev := tasks[i-1]
		curr := tasks[i]
		if err := verifyTaskCompatibility(prev, curr); err != nil {
			return nil, fmt.Errorf("failed to create task '%s': %w", config.Name, err)
		}
	}

	last := tasks[size-1]
	hasLastTaskOut := len(last.ReturnValueTypes) > 0

	if err := verifyReturnType[T](tasks[size-1]); err != nil {
		return nil, fmt.Errorf("failed to create task '%s': %w", config.Name, err)
	}

	return &TaskChain[T]{
		Tasks:     tasks,
		hasOutput: hasLastTaskOut,
		Cleanup:   config.Cleanup,
		Name:      config.Name,
	}, nil
}

func Zero[T any]() T {
	return *new(T)
}

// Run a series of Task in this TaskChain.
func (tc *TaskChain[T]) Run(ctx context.Context, params ...interface{}) (T, error) {
	if len(tc.Tasks) == 0 {
		return tc.doCleanup(Zero[T](), nil)
	}

	if err := verifyInitialParams(tc.Tasks[0].Fn.Type(), params); err != nil {
		return tc.doCleanup(Zero[T](), fmt.Errorf("error running %s: %w", tc.Name, err))
	}

	var currentResults = make([]reflect.Value, len(params))

	for i := range params {
		currentResults[i] = reflect.ValueOf(params[i])
	}

	for i, t := range tc.Tasks {
		if ctx != nil && ctx.Err() != nil {
			return tc.doCleanup(Zero[T](), ctx.Err())
		}
		results, err := t.Run(currentResults)
		if err != nil {
			return tc.doCleanup(Zero[T](), fmt.Errorf("error running task %d: %w", i, err))
		}
		currentResults = results
	}

	if len(currentResults) == 0 {
		return tc.doCleanup(Zero[T](), nil)
	}

	return tc.doCleanup(currentResults[len(currentResults)-1].Interface().(T), nil)
}

func (tc *TaskChain[T]) doCleanup(v T, err error) (T, error) {
	if tc.Cleanup != nil {
		if e := tc.Cleanup(); e != nil {
			return v, errors.Join(e, err)
		}
	}
	return v, err
}
