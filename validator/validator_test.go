package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type User struct {
	Name  string `validate:"required" json:"name"`
	Email string `validate:"required,email" json:"email"`
	Age   int    `validate:"gte=18" json:"age"`
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

		// Check if error message uses JSON tag
		errStr := GetErrorFirstStr(err)
		assert.Contains(t, errStr, "name") // Should be "name" not "Name"
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

	t.Run("GetErrors Safety", func(t *testing.T) {
		assert.Nil(t, GetErrors(nil))
		assert.Nil(t, GetErrors(assert.AnError))
	})
}
