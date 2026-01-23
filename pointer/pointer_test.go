package pointer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPointer(t *testing.T) {
	t.Run("Of", func(t *testing.T) {
		str := "hello"
		ptr := Of(str)
		assert.NotNil(t, ptr)
		assert.Equal(t, str, *ptr)

		i := 123
		iptr := Of(i)
		assert.NotNil(t, iptr)
		assert.Equal(t, i, *iptr)
	})

	t.Run("Legacy Helpers", func(t *testing.T) {
		s := "test"
		assert.Equal(t, s, *String(s))

		i := 10
		assert.Equal(t, i, *Int(i))

		b := true
		assert.Equal(t, b, *Bool(b))

		now := time.Now()
		assert.Equal(t, now, *Time(now))
	})
}
