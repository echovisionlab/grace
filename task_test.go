package grace

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNewTask(t *testing.T) {
	t.Run("must return error when func is nil", func(t *testing.T) {
		task, err := NewTask(&TaskConfig{"test", nil, nil})
		assert.Error(t, err)
		assert.Nil(t, task)
	})

	t.Run("must return not a function err", func(t *testing.T) {
		task, err := NewTask(&TaskConfig{"test", nil, 10})
		assert.Nil(t, task)
		assert.ErrorIs(t, err, NotFuncErr)
	})

	t.Run("must handle no return type", func(t *testing.T) {
		count := 0
		task, err := NewTask(&TaskConfig{"my task", nil, func() { count++ }})
		assert.NoError(t, err)
		assert.NotNil(t, task)
		v, err := task.Run(nil)
		assert.Nil(t, v)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestTask_Run(t *testing.T) {
	t.Run("must return task return value", func(t *testing.T) {
		task, err := NewTask(&TaskConfig{"test", nil, func() string { return "test_val" }})
		assert.NoError(t, err)
		v, err := task.Run(nil)
		assert.NoError(t, err)
		assert.NotNil(t, v)
		assert.Len(t, v, 1)
		val, ok := v[0].Interface().(string)
		assert.True(t, ok)
		assert.Equal(t, val, "test_val")
	})

	t.Run("must return error", func(t *testing.T) {
		expectedErr := fmt.Errorf("test error")
		fn := func() (int, error) {
			return -1, expectedErr
		}
		task, err := NewTask(&TaskConfig{"test", nil, fn})
		assert.NoError(t, err)
		v, err := task.Run(nil)
		assert.ErrorContains(t, err, expectedErr.Error())
		assert.Nil(t, v)
	})

	t.Run("must not return error", func(t *testing.T) {
		fn := func(v int) (int, int, error) {
			return v + 10, v - 10, nil
		}
		task, err := NewTask(&TaskConfig{"test", nil, fn})
		assert.NoError(t, err)
		v, err := task.Run([]reflect.Value{reflect.ValueOf(10)})
		assert.NoError(t, err)
		assert.Len(t, v, 2)

		val, ok := v[0].Interface().(int)
		assert.True(t, ok)
		assert.Equal(t, 20, val)

		val, ok = v[1].Interface().(int)
		assert.True(t, ok)
		assert.Equal(t, 0, val)
	})

	t.Run("must join Cleanup error", func(t *testing.T) {
		e1, e2 := errors.New("e1"), errors.New("e2")

		task, err := NewTask(&TaskConfig{
			"test",
			func() error { return e1 },
			func() error { return e2 }})
		assert.NoError(t, err)
		assert.NotNil(t, task)

		v, err := task.Run(nil)
		assert.ErrorIs(t, err, e1)
		assert.ErrorIs(t, err, e2)
		assert.Nil(t, v)
	})
}
