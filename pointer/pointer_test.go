package pointer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOf(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		p := Of(true)
		assert.NotNil(t, p)
		assert.True(t, *p)
	})
	t.Run("Int", func(t *testing.T) {
		p := Of(42)
		assert.NotNil(t, p)
		assert.Equal(t, 42, *p)
	})
	t.Run("String", func(t *testing.T) {
		p := Of("hello")
		assert.Equal(t, "hello", *p)
	})
	t.Run("Struct", func(t *testing.T) {
		type S struct{ Name string }
		p := Of(S{Name: "test"})
		assert.Equal(t, "test", p.Name)
	})
}

func TestString(t *testing.T) {
	p := String("hello")
	assert.Equal(t, "hello", *p)
}

func TestInt(t *testing.T) {
	p := Int(42)
	assert.Equal(t, 42, *p)
}

func TestBool(t *testing.T) {
	p := Bool(true)
	assert.True(t, *p)
	p = Bool(false)
	assert.False(t, *p)
}

func TestTime(t *testing.T) {
	now := time.Now()
	p := Time(now)
	assert.Equal(t, now, *p)
}

func TestDeref(t *testing.T) {
	t.Run("Non-nil string", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", Deref(&s))
	})
	t.Run("Nil string", func(t *testing.T) {
		assert.Equal(t, "", Deref[string](nil))
	})
	t.Run("Non-nil int", func(t *testing.T) {
		i := 42
		assert.Equal(t, 42, Deref(&i))
	})
	t.Run("Nil int", func(t *testing.T) {
		assert.Equal(t, 0, Deref[int](nil))
	})
	t.Run("Non-nil bool", func(t *testing.T) {
		b := true
		assert.True(t, Deref(&b))
	})
	t.Run("Nil bool", func(t *testing.T) {
		assert.False(t, Deref[bool](nil))
	})
}

func TestDerefOr(t *testing.T) {
	t.Run("Non-nil uses value", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", DerefOr(&s, "default"))
	})
	t.Run("Nil uses default", func(t *testing.T) {
		assert.Equal(t, "default", DerefOr[string](nil, "default"))
	})
	t.Run("Nil int uses default", func(t *testing.T) {
		assert.Equal(t, 42, DerefOr[int](nil, 42))
	})
	t.Run("Non-nil int uses value", func(t *testing.T) {
		i := 10
		assert.Equal(t, 10, DerefOr(&i, 42))
	})
	t.Run("Zero value pointer returns zero", func(t *testing.T) {
		i := 0
		assert.Equal(t, 0, DerefOr(&i, 42))
	})
}
