package grace

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestNewTaskChain(t *testing.T) {
	noOpCleanFn := func() error { return nil }
	first, err := NewTask(&TaskConfig{
		Name:    "first",
		Cleanup: nil,
		Fn: func(s string) (int, error) {
			return strconv.Atoi(s)
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, first)

	second, err := NewTask(&TaskConfig{
		Name:    "second",
		Cleanup: nil,
		Fn: func(n int) int {
			return n * 10
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, second)

	t.Run("must filter nil tasks", func(t *testing.T) {
		chain, err := NewTaskChain[struct{}](&TaskChainConfig{
			Name:    "",
			Cleanup: nil,
			Tasks:   []*Task{nil, nil, nil},
		})
		assert.NoError(t, err)
		assert.NotNil(t, chain)
		v, err := chain.Run(nil)
		assert.NoError(t, err)
		assert.Zero(t, v)
		assert.NotNil(t, v)
	})

	t.Run("must report invalid return type", func(t *testing.T) {
		chain, err := NewTaskChain[struct{}](&TaskChainConfig{
			"test",
			noOpCleanFn,
			[]*Task{nil, second},
		})
		assert.ErrorContains(t, err, "struct {}")
		assert.ErrorContains(t, err, "int")
		assert.Nil(t, chain)
	})

	t.Run("must report output and input count mismatch", func(t *testing.T) {
		t1, _ := NewTask(&TaskConfig{"test1", nil, func() {}})
		t2, _ := NewTask(&TaskConfig{"test2", nil, func(a, b int) int { return 0 }})
		chain, err := NewTaskChain[int](&TaskChainConfig{"chain", noOpCleanFn, []*Task{t1, t2}})
		assert.ErrorContains(t, err, "2")
		assert.ErrorContains(t, err, "1")
		assert.Nil(t, chain)
	})

	t.Run("must report output and input type mismatch", func(t *testing.T) {
		t1, _ := NewTask(&TaskConfig{"test1", nil, func() int { return 0 }})
		t2, _ := NewTask(&TaskConfig{"test2", nil, func(s string) int { return 0 }})
		chain, err := NewTaskChain[int](&TaskChainConfig{"chain", noOpCleanFn, []*Task{t1, t2}})
		assert.ErrorContains(t, err, "string")
		assert.ErrorContains(t, err, "int")
		assert.Nil(t, chain)
	})

	t.Run("must handle no output func", func(t *testing.T) {
		x := 0
		t1, _ := NewTask(&TaskConfig{"t1", nil, func(a, b, c int) int { return a + b*c }})
		t2, _ := NewTask(&TaskConfig{"t2", nil, func(i int) { x = i * 10 }})
		chain, err := NewTaskChain[int](&TaskChainConfig{"test", noOpCleanFn, []*Task{t1, t2}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)
		r, err := chain.Run(context.Background(), 10, 3, 5)
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, 250, x)
	})

	t.Run("must handle error from chain", func(t *testing.T) {
		expectedErr := errors.New("my test error")
		t1, err := NewTask(&TaskConfig{"t1", nil, func(a, b, c int) (int, error) { return a + b + c, nil }})
		assert.NoError(t, err)
		assert.NotNil(t, t1)

		t2, err := NewTask(&TaskConfig{"t2", nil, func(a int) (string, error) { return strconv.Itoa(a), expectedErr }})

		invoked := 0
		t3, err := NewTask(&TaskConfig{"t3", nil, func(s string) { invoked++ }})

		chain, err := NewTaskChain[any](&TaskChainConfig{"test_chain", nil, []*Task{t1, t2, t3}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(context.Background(), 1, 2, 3)
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, v)
	})

	t.Run("must return value", func(t *testing.T) {
		chain, err := NewTaskChain[int](&TaskChainConfig{"test_chain", nil, []*Task{first, second}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)
		v, err := chain.Run(context.Background(), "10")
		assert.NoError(t, err)
		assert.Equal(t, 100, v)
	})

	t.Run("must run with nil context", func(t *testing.T) {
		chain, err := NewTaskChain[int](&TaskChainConfig{"test_chain", nil, []*Task{first, second}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)
		v, err := chain.Run(nil, "10")
		assert.NoError(t, err)
		assert.Equal(t, 100, v)
	})

	t.Run("must exit on context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		c1, c2 := 0, 0
		t1, _ := NewTask(&TaskConfig{"t1", nil, func() { cancel(); c1++ }})
		t2, _ := NewTask(&TaskConfig{"t2", nil, func() { c2++ }})

		chain, err := NewTaskChain[any](&TaskChainConfig{"test_chain", nil, []*Task{t1, t2}})
		assert.NoError(t, err)

		v, err := chain.Run(ctx)
		assert.Nil(t, v)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, c1)
		assert.Equal(t, 0, c2)
	})

	t.Run("must exit immediately on context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		delay := time.Millisecond * 100
		c1, c2 := 0, 0
		t1, _ := NewTask(&TaskConfig{"t1", nil, func() { c1++; time.Sleep(delay) }})
		t2, _ := NewTask(&TaskConfig{"t2", nil, func() { c2++ }})

		chain, err := NewTaskChain[any](&TaskChainConfig{"test_chain", nil, []*Task{t1, t2}})
		assert.NoError(t, err)

		begin := time.Now()
		v, err := chain.Run(ctx)
		took := time.Now().Sub(begin)
		assert.Greater(t, took, delay)
		assert.Nil(t, v)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 1, c1)
		assert.Equal(t, 0, c2)
	})

	t.Run("must run Cleanup", func(t *testing.T) {
		count := 0
		chain, err := NewTaskChain[int](&TaskChainConfig{
			"test",
			func() error {
				count++
				return nil
			},
			[]*Task{first, second}})

		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(nil, "30")
		assert.NoError(t, err)
		assert.Equal(t, 300, v)

		assert.Equal(t, 1, count)
	})

	t.Run("must join error from Cleanup", func(t *testing.T) {
		count := 0
		testErr := errors.New("test error")
		cleanupErr := errors.New("Cleanup error")
		t1, _ := NewTask(&TaskConfig{"test task", nil, func() error { count++; return testErr }})
		chain, err := NewTaskChain[any](
			&TaskChainConfig{"test task chain", func() error {
				count++
				return cleanupErr
			}, []*Task{t1}})

		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(nil)
		assert.Nil(t, v)
		assert.ErrorIs(t, err, testErr)
		assert.ErrorIs(t, err, cleanupErr)
	})

	t.Run("must return error when args size does not match", func(t *testing.T) {
		t1, _ := NewTask(&TaskConfig{"test_task", nil, func(a, b, c int) {}})
		chain, err := NewTaskChain[any](&TaskChainConfig{"test_task_chain", nil, []*Task{t1}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(context.Background(), 1, 2, 3, 4)
		assert.ErrorContains(t, err, "4")
		assert.ErrorContains(t, err, "3")
		assert.ErrorContains(t, err, "test_task_chain")
		assert.Nil(t, v)
	})

	t.Run("must return error when args type does not match", func(t *testing.T) {
		t1, _ := NewTask(&TaskConfig{"test_task", nil, func(a, b, c int) {}})
		chain, err := NewTaskChain[any](&TaskChainConfig{"test_task_chain", nil, []*Task{t1}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(context.Background(), 1, "2", 3)
		assert.ErrorContains(t, err, "2")
		assert.ErrorContains(t, err, "string")
		assert.ErrorContains(t, err, "int")
		assert.ErrorContains(t, err, "test_task_chain")
		assert.Nil(t, v)
	})

	t.Run("must run all the Cleanup functions from the tasks", func(t *testing.T) {
		var a, b, c, d, e int
		var x int
		testErr := errors.New("test error")
		t1, _ := NewTask(&TaskConfig{"t1", func() error { a += 1; return nil }, func() {}})
		t2, _ := NewTask(&TaskConfig{"t2", func() error { b += 2; return nil }, func() {}})
		t3, _ := NewTask(&TaskConfig{"t3", func() error { c += 3; return nil }, func() {}})
		t4, _ := NewTask(&TaskConfig{"t4", func() error { d += 4; return testErr }, func() {}})
		t5, _ := NewTask(&TaskConfig{"t5", func() error { e += 5; return nil }, func() {}})

		chain, err := NewTaskChain[any](&TaskChainConfig{
			"test_task_chain",
			func() error { x++; return nil },
			[]*Task{t1, t2, t3, t4, t5},
		})
		assert.NoError(t, err)
		v, err := chain.Run(context.Background())
		assert.Nil(t, v)
		assert.ErrorIs(t, err, testErr)
		assert.Equal(t, 1, a)
		assert.Equal(t, 2, b)
		assert.Equal(t, 3, c)
		assert.Equal(t, 4, d)
		assert.Equal(t, 0, e)
		assert.Equal(t, 1, x)
	})

	t.Run("must run Cleanup when context err", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		var a, b, c int
		t1, _ := NewTask(&TaskConfig{"t1", func() error { a += 1; return nil }, func() { time.Sleep(time.Millisecond * 50) }})
		t2, _ := NewTask(&TaskConfig{"t2", func() error { b += 2; return nil }, func() { time.Sleep(time.Millisecond * 100) }})
		t3, _ := NewTask(&TaskConfig{"t3", func() error { c += 3; return nil }, func() { time.Sleep(time.Millisecond * 80) }})

		chain, err := NewTaskChain[any](&TaskChainConfig{"test_task_chain", nil, []*Task{t1, t2, t3}})
		assert.NoError(t, err)
		assert.NotNil(t, chain)

		v, err := chain.Run(ctx)
		assert.Nil(t, v)
		assert.ErrorIs(t, err, context.DeadlineExceeded)

		assert.Equal(t, 1, a)
		assert.Equal(t, 2, b)
		assert.Equal(t, 0, c)
	})
}
