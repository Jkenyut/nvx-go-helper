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

	t.Run("Nil error returns empty", func(t *testing.T) {
		assert.Equal(t, "", GetErrorFirstStr(nil))
		assert.Equal(t, "", GetErrorsFullStr(nil))
		assert.Equal(t, "", GetErrorFirstMsg(nil))
		assert.Equal(t, "", GetErrorsFullMsg(nil))
	})

	t.Run("RegisterCustomValidation", func(t *testing.T) {
		err := RegisterCustomValidation("is_awesome", func(fl v10.FieldLevel) bool {
			return fl.Field().String() == "awesome"
		}, "Must be awesome exactly like %s")
		assert.NoError(t, err)

		type TestCustom struct {
			Title string `validate:"is_awesome=Gojek" json:"title"`
		}

		err = Struct(TestCustom{Title: "awesome"})
		assert.NoError(t, err)

		err = Struct(TestCustom{Title: "boring"})
		assert.Error(t, err)
		assert.Contains(t, GetErrorFirstMsg(err), "title: Must be awesome exactly like Gojek")
	})

	t.Run("RegisterCustomValidation_NoParam", func(t *testing.T) {
		err := RegisterCustomValidation("is_cool", func(fl v10.FieldLevel) bool {
			return fl.Field().String() == "cool"
		}, "Must be cool")
		assert.NoError(t, err)

		type TestStruct struct {
			Status string `validate:"is_cool" json:"status"`
		}

		err = Struct(TestStruct{Status: "not cool"})
		assert.Error(t, err)
		assert.Equal(t, "status: Must be cool", GetErrorFirstMsg(err))
	})
}

func TestValidatorInterface(t *testing.T) {
	val := New()
	assert.NotNil(t, val)
	assert.NotNil(t, val.Raw())

	err := val.RegisterCustomValidation("is_magic", func(fl v10.FieldLevel) bool {
		return fl.Field().String() == "abracadabra"
	}, "Must recite the magic word %s")
	assert.NoError(t, err)

	type Spell struct {
		Word string `validate:"is_magic=please" json:"word"`
	}

	err = val.Struct(Spell{Word: "hello"})
	assert.Error(t, err)
	assert.Equal(t, "word: Must recite the magic word please", val.GetErrorFirstMsg(err))

	err = val.Struct(Spell{Word: "abracadabra"})
	assert.NoError(t, err)
}

func TestAllValidationTags(t *testing.T) {
	val := New()

	t.Run("min", func(t *testing.T) {
		type S struct {
			V string `validate:"min=3" json:"v"`
		}
		err := val.Struct(S{V: "ab"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Minimum length or value is 3")
	})

	t.Run("max", func(t *testing.T) {
		type S struct {
			V string `validate:"max=3" json:"v"`
		}
		err := val.Struct(S{V: "abcde"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Maximum length or value is 3")
	})

	t.Run("len", func(t *testing.T) {
		type S struct {
			V string `validate:"len=3" json:"v"`
		}
		err := val.Struct(S{V: "ab"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Length must be exactly 3")
	})

	t.Run("lte", func(t *testing.T) {
		type S struct {
			V int `validate:"lte=10" json:"v"`
		}
		err := val.Struct(S{V: 20})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be less than or equal to 10")
	})

	t.Run("gt", func(t *testing.T) {
		type S struct {
			V int `validate:"gt=10" json:"v"`
		}
		err := val.Struct(S{V: 5})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be greater than 10")
	})

	t.Run("lt", func(t *testing.T) {
		type S struct {
			V int `validate:"lt=10" json:"v"`
		}
		err := val.Struct(S{V: 20})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be less than 10")
	})

	t.Run("url", func(t *testing.T) {
		type S struct {
			V string `validate:"url" json:"v"`
		}
		err := val.Struct(S{V: "not-a-url"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Invalid URL format")
	})

	t.Run("uuid", func(t *testing.T) {
		type S struct {
			V string `validate:"uuid" json:"v"`
		}
		err := val.Struct(S{V: "not-a-uuid"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Invalid UUID format")
	})

	t.Run("alphanum", func(t *testing.T) {
		type S struct {
			V string `validate:"alphanum" json:"v"`
		}
		err := val.Struct(S{V: "abc!123"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Can only contain alphanumeric characters")
	})

	t.Run("alpha", func(t *testing.T) {
		type S struct {
			V string `validate:"alpha" json:"v"`
		}
		err := val.Struct(S{V: "abc123"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Can only contain alphabetic characters")
	})

	t.Run("numeric", func(t *testing.T) {
		type S struct {
			V string `validate:"numeric" json:"v"`
		}
		err := val.Struct(S{V: "abc"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid numeric value")
	})

	t.Run("boolean", func(t *testing.T) {
		type S struct {
			V string `validate:"boolean" json:"v"`
		}
		err := val.Struct(S{V: "maybe"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a boolean value")
	})

	t.Run("contains", func(t *testing.T) {
		type S struct {
			V string `validate:"contains=hello" json:"v"`
		}
		err := val.Struct(S{V: "world"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must contain the text 'hello'")
	})

	t.Run("excludes", func(t *testing.T) {
		type S struct {
			V string `validate:"excludes=bad" json:"v"`
		}
		err := val.Struct(S{V: "this is bad"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must not contain the text 'bad'")
	})

	t.Run("startswith", func(t *testing.T) {
		type S struct {
			V string `validate:"startswith=hello" json:"v"`
		}
		err := val.Struct(S{V: "world"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must start with 'hello'")
	})

	t.Run("endswith", func(t *testing.T) {
		type S struct {
			V string `validate:"endswith=world" json:"v"`
		}
		err := val.Struct(S{V: "hello"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must end with 'world'")
	})

	t.Run("eq", func(t *testing.T) {
		type S struct {
			V string `validate:"eq=exact" json:"v"`
		}
		err := val.Struct(S{V: "wrong"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be equal to exact")
	})

	t.Run("ne", func(t *testing.T) {
		type S struct {
			V string `validate:"ne=forbidden" json:"v"`
		}
		err := val.Struct(S{V: "forbidden"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must not be equal to forbidden")
	})

	t.Run("ip", func(t *testing.T) {
		type S struct {
			V string `validate:"ip" json:"v"`
		}
		err := val.Struct(S{V: "not-an-ip"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid IP address")
	})

	t.Run("base64", func(t *testing.T) {
		type S struct {
			V string `validate:"base64" json:"v"`
		}
		err := val.Struct(S{V: "not!!base64"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid Base64 string")
	})

	t.Run("json", func(t *testing.T) {
		type S struct {
			V string `validate:"json" json:"v"`
		}
		err := val.Struct(S{V: "not json"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid JSON string")
	})

	t.Run("datetime", func(t *testing.T) {
		type S struct {
			V string `validate:"datetime=2006-01-02" json:"v"`
		}
		err := val.Struct(S{V: "not-a-date"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid datetime format")
	})

	t.Run("eqfield", func(t *testing.T) {
		type S struct {
			Password        string `validate:"required" json:"password"`
			ConfirmPassword string `validate:"eqfield=Password" json:"confirm_password"`
		}
		err := val.Struct(S{Password: "abc", ConfirmPassword: "xyz"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must match Password field")
	})

	t.Run("containsany", func(t *testing.T) {
		type S struct {
			V string `validate:"containsany=!@#" json:"v"`
		}
		err := val.Struct(S{V: "plain"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must contain at least one of the following characters")
	})

	t.Run("excludesall", func(t *testing.T) {
		type S struct {
			V string `validate:"excludesall=!@#" json:"v"`
		}
		err := val.Struct(S{V: "has!mark"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must not contain any of the following characters")
	})

	t.Run("number", func(t *testing.T) {
		type S struct {
			V string `validate:"number" json:"v"`
		}
		err := val.Struct(S{V: "abc"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid number")
	})
}

func TestIndonesianValidations(t *testing.T) {
	val := New()

	t.Run("NIK Valid", func(t *testing.T) {
		type S struct {
			NIK string `validate:"nik" json:"nik"`
		}
		err := val.Struct(S{NIK: "3201010101010001"})
		assert.NoError(t, err)
	})

	t.Run("NIK Invalid", func(t *testing.T) {
		type S struct {
			NIK string `validate:"nik" json:"nik"`
		}
		err := val.Struct(S{NIK: "123"})
		assert.Error(t, err)
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid Indonesian NIK")
	})

	t.Run("NPWP Valid", func(t *testing.T) {
		type S struct {
			NPWP string `validate:"npwp" json:"npwp"`
		}
		err := val.Struct(S{NPWP: "123456789012345"})
		assert.NoError(t, err)
	})

	t.Run("NPWP Valid with dots/dashes", func(t *testing.T) {
		type S struct {
			NPWP string `validate:"npwp" json:"npwp"`
		}
		err := val.Struct(S{NPWP: "12.345.678.9-012.345"})
		assert.NoError(t, err)
	})

	t.Run("NPWP Invalid", func(t *testing.T) {
		type S struct {
			NPWP string `validate:"npwp" json:"npwp"`
		}
		err := val.Struct(S{NPWP: "123"})
		assert.Error(t, err)
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid Indonesian NPWP")
	})

	t.Run("Phone ID Valid +62", func(t *testing.T) {
		type S struct {
			Phone string `validate:"phone_id" json:"phone"`
		}
		err := val.Struct(S{Phone: "+6281234567890"})
		assert.NoError(t, err)
	})

	t.Run("Phone ID Valid 08", func(t *testing.T) {
		type S struct {
			Phone string `validate:"phone_id" json:"phone"`
		}
		err := val.Struct(S{Phone: "081234567890"})
		assert.NoError(t, err)
	})

	t.Run("Phone ID Invalid", func(t *testing.T) {
		type S struct {
			Phone string `validate:"phone_id" json:"phone"`
		}
		err := val.Struct(S{Phone: "123"})
		assert.Error(t, err)
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid Indonesian phone number")
	})

	t.Run("nefield", func(t *testing.T) {
		type S struct {
			Old string `validate:"required" json:"old"`
			New string `validate:"nefield=Old" json:"new"`
		}
		err := val.Struct(S{Old: "same", New: "same"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must not match Old field")
	})

	t.Run("mac", func(t *testing.T) {
		type S struct {
			V string `validate:"mac" json:"v"`
		}
		err := val.Struct(S{V: "not-a-mac"})
		assert.Contains(t, val.GetErrorFirstMsg(err), "Must be a valid MAC address")
	})

	t.Run("Unknown tag fallback", func(t *testing.T) {
		err := val.Raw().RegisterValidation("custom_xyz", func(fl v10.FieldLevel) bool {
			return false
		})
		assert.NoError(t, err)

		type S struct {
			V string `validate:"custom_xyz" json:"v"`
		}
		verr := val.Struct(S{V: "test"})
		msg := val.GetErrorFirstMsg(verr)
		assert.Contains(t, msg, "Failed on custom_xyz validation")
	})
}

func TestGetErrorsFullStr(t *testing.T) {
	type S struct {
		Name  string `validate:"required" json:"name"`
		Email string `validate:"required,email" json:"email"`
	}
	err := Struct(S{})
	fullStr := GetErrorsFullStr(err)
	assert.Contains(t, fullStr, "name: required")
	assert.Contains(t, fullStr, "email: required")
}
