package validator

import (
	"fmt"
	"testing"

	v10 "github.com/go-playground/validator/v10"
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
	t.Run("GetErrorFirstMsg", func(t *testing.T) {
		type TestStruct struct {
			Email string `validate:"required,email" json:"email"`
			Age   int    `validate:"gte=18" json:"age"`
			Role  string `validate:"oneof=admin user" json:"role"`
		}

		// Test required
		err := Struct(TestStruct{Email: "", Age: 20, Role: "admin"})
		assert.Equal(t, "email: This field is required", GetErrorFirstMsg(err))

		// Test email format
		err = Struct(TestStruct{Email: "not-an-email", Age: 20, Role: "admin"})
		assert.Equal(t, "email: Invalid email address format", GetErrorFirstMsg(err))

		// Test gte
		err = Struct(TestStruct{Email: "budi@example.com", Age: 17, Role: "admin"})
		assert.Equal(t, "age: Must be greater than or equal to 18", GetErrorFirstMsg(err))

		// Test oneof
		err = Struct(TestStruct{Email: "budi@example.com", Age: 20, Role: "guest"})
		assert.Equal(t, "role: Must be one of [admin, user]", GetErrorFirstMsg(err))
	})

	t.Run("GetErrorsFullMsg", func(t *testing.T) {
		type TestStruct struct {
			Email string `validate:"required,email" json:"email"`
			Age   int    `validate:"gte=18" json:"age"`
		}

		err := Struct(TestStruct{Email: "not-an-email", Age: 10})
		// The error should contain both translations
		fullMsg := GetErrorsFullMsg(err)
		assert.Contains(t, fullMsg, "email: Invalid email address format")
		assert.Contains(t, fullMsg, "age: Must be greater than or equal to 18")

		// Test Fallback when it's not a validation error
		genericErr := fmt.Errorf("database timeout")
		assert.Equal(t, "database timeout", GetErrorFirstStr(genericErr))
		assert.Equal(t, "database timeout", GetErrorsFullStr(genericErr))
		assert.Equal(t, "database timeout", GetErrorFirstMsg(genericErr))
		assert.Equal(t, "database timeout", GetErrorsFullMsg(genericErr))
	})

	t.Run("RegisterCustomValidation", func(t *testing.T) {
		// Register a fake custom tag with a parameter message
		err := RegisterCustomValidation("is_awesome", func(fl v10.FieldLevel) bool {
			return fl.Field().String() == "awesome"
		}, "Must be awesome exactly like %s")
		assert.NoError(t, err)

		type TestCustom struct {
			Title string `validate:"is_awesome=Gojek" json:"title"`
		}

		// Valid case
		err = Struct(TestCustom{Title: "awesome"})
		assert.NoError(t, err)

		// Invalid case
		err = Struct(TestCustom{Title: "boring"})
		assert.Error(t, err)
		assert.Contains(t, GetErrorFirstMsg(err), "title: Must be awesome exactly like Gojek")
	})
}

func TestValidatorInterface(t *testing.T) {
	// Create a new fresh instance
	val := New()
	assert.NotNil(t, val)

	// Ensure Raw() returns the underlying *v10.Validate
	assert.NotNil(t, val.Raw())

	// Register a custom validation on this specific instance
	err := val.RegisterCustomValidation("is_magic", func(fl v10.FieldLevel) bool {
		return fl.Field().String() == "abracadabra"
	}, "Must recite the magic word %s")
	assert.NoError(t, err)

	type Spell struct {
		Word string `validate:"is_magic=please" json:"word"`
	}

	// Test invalid
	err = val.Struct(Spell{Word: "hello"})
	assert.Error(t, err)
	assert.Equal(t, "word: Must recite the magic word please", val.GetErrorFirstMsg(err))

	// Test valid
	err = val.Struct(Spell{Word: "abracadabra"})
	assert.NoError(t, err)
}
