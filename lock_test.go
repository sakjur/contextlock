package contextlock_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/sakjur/contextlock"
)

func Equal[T any](t testing.TB, expected, actual T) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected '%v' == got '%v'\n", expected, actual)
		t.FailNow()
	}
}

func True(t testing.TB, actual bool) {
	t.Helper()

	if !actual {
		t.Errorf("expected true, got '%v'\n", actual)
		t.FailNow()
	}
}

func False(t testing.TB, actual bool) {
	t.Helper()

	if actual {
		t.Errorf("expected false, got %v\n", actual)
		t.FailNow()
	}
}

func Nil(t testing.TB, got any) {
	t.Helper()

	if got != nil {
		t.Errorf("expected <nil> == got '%v'\n", got)
		t.FailNow()
	}
}

func TestWithValue(t *testing.T) {
	const key = "key"
	const value = "value"
	const lock = "lås"

	ctx := context.Background()
	ctx = contextlock.WithValue(ctx, lock, key, value)

	// Validate that the type of the new value is a
	// contextlock.Container.
	container := ctx.Value(key)
	Equal(t, reflect.TypeOf(contextlock.Container{}), reflect.TypeOf(container))

	// the lock is locked by default.
	v, ok := container.(contextlock.Container).Value(ctx)
	False(t, ok)
	Nil(t, v)

	// Create a new context.Context where context is unlocked.
	ctx2 := contextlock.Unlock(ctx, lock)

	// the lock in the new context is unlocked.
	v, ok = contextlock.Value(ctx2, key)
	True(t, ok)
	Equal(t, value, v)

	// the lock in the original context is still locked.
	v, ok = contextlock.Value(ctx, key)
	False(t, ok)
	Nil(t, v)
}

func TestDifferentLocks(t *testing.T) {
	type lockA struct{}
	type lockB struct{}

	const key = "key"
	const value = "value"

	ctx := context.Background()
	ctx = contextlock.WithValue(ctx, lockA{}, key, value)

	// unlocking the other lock doesn't unlock our container.
	ctx = contextlock.Unlock(ctx, lockB{})
	v, ok := contextlock.Value(ctx, key)
	False(t, ok)
	Nil(t, v)

	// unlocking our lock unlocks our container.
	ctx = contextlock.Unlock(ctx, lockA{})
	v, ok = contextlock.Value(ctx, key)
	True(t, ok)
	Equal(t, value, v)
}

func TestTimeLock(t *testing.T) {
	t0 := time.Date(2007, 8, 1, 15, 0, 0, 0, time.UTC)
	tNow := t0
	nowFn := func() time.Time { return tNow }

	type lock struct{}

	key := lock{}
	ctx := contextlock.TimeLock(
		context.Background(),
		key,
		t0.Add(time.Hour),
		contextlock.TimeSource(nowFn),
	)

	tests := []struct {
		testTime time.Time
		unlocked bool
	}{
		{time.Time{}, false},
		{t0, false},
		{t0.Add(time.Hour), false},
		{t0.Add(time.Hour + time.Nanosecond), true},
		{t0.Add(525600 * time.Minute), true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s = %v", tc.testTime, tc.unlocked), func(t *testing.T) {
			tNow = tc.testTime
			Equal(t, tc.unlocked, contextlock.Unlocked(ctx, key))
		})
	}
}

func TestFunctionLock(t *testing.T) {
	type lock struct{}
	type otherLock struct{}
	key := lock{}
	otherKey := otherLock{}

	tests := []struct {
		name     string
		parent   context.Context
		fn       func(ctx context.Context) bool
		unlocked bool
	}{
		{
			name:   "returns false",
			parent: context.Background(),
			fn: func(ctx context.Context) bool {
				return false
			},
			unlocked: false,
		},
		{
			name:   "returns true",
			parent: context.Background(),
			fn: func(ctx context.Context) bool {
				return true
			},
			unlocked: true,
		},
		{
			name:   "returns true if another lock is unlocked (locked)",
			parent: context.Background(),
			fn: func(ctx context.Context) bool {
				return contextlock.Unlocked(ctx, otherKey)
			},
			unlocked: false,
		},
		{
			name:   "returns true if another lock is unlocked (unlocked)",
			parent: contextlock.Unlock(context.Background(), otherKey),
			fn: func(ctx context.Context) bool {
				return contextlock.Unlocked(ctx, otherKey)
			},
			unlocked: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := contextlock.FunctionLock(tc.parent, key, tc.fn)
			Equal(t, tc.unlocked, contextlock.Unlocked(ctx, key))
		})
	}
}
