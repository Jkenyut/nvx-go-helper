package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type User struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=18"`
}

func TestValidator(t *testing.T) {
	t.Run("Struct Valid", func(t *testing.T) {
		user := User{Name: "Budi", Email: "budi@example.com", Age: 20}
		err := Struct(user)
		assert.NoError(t, err)
	})

	t.Run("Struct Invalid", func(t *testing.T) {
		user := User{Name: "", Email: "bad-email", Age: 10}
		err := Struct(user)
		assert.Error(t, err)
	})

	t.Run("Var Valid", func(t *testing.T) {
		err := Var("test@example.com", "email")
		assert.NoError(t, err)
	})

	t.Run("Var Invalid", func(t *testing.T) {
		err := Var("not-email", "email")
		assert.Error(t, err)
	})

	t.Run("Singleton", func(t *testing.T) {
		v1 := Get()
		v2 := Get()
		assert.Equal(t, v1, v2)
	})
}
