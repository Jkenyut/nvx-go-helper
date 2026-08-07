package cryptoutil

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUIDV4(t *testing.T) {
	t.Run("V4 String", func(t *testing.T) {
		id1 := V4()
		id2 := V4()
		assert.NotEmpty(t, id1)
		assert.NotEqual(t, id1, id2)

		parsed, err := uuid.Parse(id1)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Version(4), parsed.Version())
	})

	t.Run("V4UUID Object", func(t *testing.T) {
		id := V4UUID()
		assert.NotEqual(t, uuid.Nil, id)
		assert.Equal(t, uuid.Version(4), id.Version())
	})
}

func TestUUIDV7(t *testing.T) {
	t.Run("V7 String", func(t *testing.T) {
		id1 := V7()
		time.Sleep(1 * time.Millisecond) // Ensure monotonic time difference
		id2 := V7()

		assert.NotEmpty(t, id1)
		assert.NotEqual(t, id1, id2)

		// V7 roughly sorts by time
		assert.Less(t, id1, id2)

		parsed, err := uuid.Parse(id1)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Version(7), parsed.Version())
	})

	t.Run("V7UUID Object", func(t *testing.T) {
		id := V7UUID()
		assert.NotEqual(t, uuid.Nil, id)
		assert.Equal(t, uuid.Version(7), id.Version())
	})
}

func TestUUIDValidation(t *testing.T) {
	validUUID := "501438f4-2c63-42e8-b789-29158fbbe578"
	invalidUUID := "not-a-uuid"

	t.Run("Parse", func(t *testing.T) {
		u := Parse(validUUID)
		assert.Equal(t, validUUID, u.String())

		uInvalid := Parse(invalidUUID)
		assert.Equal(t, uuid.Nil, uInvalid)
	})

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, IsValid(validUUID))
		assert.True(t, IsValid("501438f42c6342e8b78929158fbbe578")) // standard hex
		assert.False(t, IsValid(invalidUUID))
		assert.False(t, IsValid(""))
	})
}
