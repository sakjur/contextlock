// SPDX-License-Identifier: MIT-0

package contextlock

import (
	"context"
	"time"
)

// A Container is a wrapper for any type and a lock.
//
// This type is used for any key-value pairs added to a
// [context.Context] using [WithValue]. Cannot be initialized from
// outside the contextlock package.
type Container struct {
	key   lock
	value any
}

// lock wraps a key to ensure that a lock can only be unlocked from
// functions in the contextlock package.
type lock any

// timestamp combines a [time.Time] with a function that returns a
// time.Time for the current time to allow overriding [time.Now].
type timestamp struct {
	Time       time.Time
	TimeSource func() time.Time
}

// TimestampOption provides functional options for a [TimeLock].
type TimestampOption func(timestamp) timestamp

type lockFunction func(ctx context.Context) bool

// Unlock returns a copy of parent where the lock behind lockKey is
// unlocked.
func Unlock(parent context.Context, lockKey any) context.Context {
	return context.WithValue(parent, lock(lockKey), true)
}

// Lock returns a copy of parent where the lock behind lockKey is
// locked.
//
// Locks do not need to be initialized and will remain locked until
// explicitly unlocked, calling this function is only necessary if you
// want to lock a previously unlocked lock.
func Lock(parent context.Context, lockKey any) context.Context {
	return context.WithValue(parent, lock(lockKey), false)
}

// TimeLock returns a copy of parent where the lock will be unlocked
// at a provided point in time.
func TimeLock(parent context.Context, lockKey any, t time.Time, opts ...TimestampOption) context.Context {
	ts := timestamp{Time: t, TimeSource: time.Now}
	for _, o := range opts {
		ts = o(ts)
	}

	return context.WithValue(parent, lock(lockKey), ts)
}

// FunctionLock returns a copy of parent where the lock calls fn to
// check whether it's unlocked.
//
// The lock is unlocked when fn returns true and locked when fn returns
// false. The context at the time of calling [Unlocked] will be passed
// as the sole argument to the fn function. The fn function must have
// the following signature:
//
//	fn(ctx context.Context) bool
func FunctionLock(parent context.Context, lockKey any, fn lockFunction) context.Context {
	return context.WithValue(parent, lock(lockKey), fn)
}

// TimeSource can be passed as a functional option to [TimeLock] to
// override [time.Now] when checking whether a lock is open or not.
func TimeSource(fn func() time.Time) TimestampOption {
	return func(t timestamp) timestamp {
		t.TimeSource = fn
		return t
	}
}

// Unlocked returns true if the lock behind lockKey in ctx is unlocked.
func Unlocked(ctx context.Context, lockKey any) bool {
	switch val := ctx.Value(lock(lockKey)).(type) {
	case bool:
		return val
	case timestamp:
		return val.Time.Before(val.TimeSource())
	case lockFunction:
		return val(ctx)
	default:
		return false
	}
}

// WithValue returns a copy of parent in which the key is associated
// with a [Container] containing the value behind a lockKey.
//
// This works like [context.WithValue] except the value is always of
// type [Container] and will refuse to return the value until the
// lockKey has been unlocked with [Unlock].
func WithValue(parent context.Context, lockKey, key, value any) context.Context {
	return context.WithValue(parent, key, Container{
		key:   lock(lockKey),
		value: value,
	})
}

// Value returns the value contained in the container if and only if
// the container's lock in ctx is unlocked.
//
// The first value is the item stored in the container, or nil if the
// lock is locked. The second value returned is a boolean which is false
// if the container is locked and true otherwise.
func (c Container) Value(ctx context.Context) (any, bool) {
	if !Unlocked(ctx, c.key) {
		return nil, false
	}

	return c.value, true
}

// Value returns the stored value for the given key. If the key is a
// [Container] added with [WithValue], the container will be unwrapped
// and [Container.Value] will be called.
//
// If the value is not a container, (value, false) is returned.
// If the value is a container and the lock is locked, (nil, false) is
// returned.
// If the value is a container and the lock is unlocked, (value, true)
// is returned.
//
// Calls [context.Context.Value] for the ctx with the given key.
func Value(ctx context.Context, key any) (any, bool) {
	value := ctx.Value(key)
	container, ok := value.(Container)
	if !ok {
		// the value is not protected by the contextlock package,
		// we return it but also false to indicate that there wasn't
		// any lock present.
		return value, false
	}

	return container.Value(ctx)
}
