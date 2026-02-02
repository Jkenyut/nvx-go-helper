// Package validator provides a singleton instance of go-playground/validator
// and wrapper functions to simplify struct validation.
//
// It ensures consistent validation rules across the application.
package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

// Get returns the singleton validator instance.
// It ensures that the validator cache is clear and built once (thread-safe).
func Get() *validator.Validate {
	// once.Do guarantees the function is called exactly once,
	// even if called concurrently from multiple goroutines.
	once.Do(func() {
		validate = validator.New()

		// RegisterTagNameFunc registers a function to get the tag name from the struct field.
		// This is used to return the JSON tag name in validation errors.
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return validate
}

// Struct validates a struct and returns the first error encountered, or nil.
//
// Example:
//
//	err := validator.Struct(req)
func Struct(s any) error {
	return Get().Struct(s)
}

// Var validates a single variable.
//
// Example:
//
//	err := validator.Var(email, "required,email")
func Var(field any, tag string) error {
	return Get().Var(field, tag)
}

// GetErrors returns the validation errors from a validator error.
func GetErrors(err error) validator.ValidationErrors {
	if err == nil {
		return nil
	}
	if errs, ok := err.(validator.ValidationErrors); ok {
		return errs
	}
	return nil
}

// GetErrorsFullStr returns the full validation errors from a validator error.
func GetErrorsFullStr(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return ""
	}

	if len(errors) == 0 {
		return ""
	}

	var result = make([]string, len(errors))
	for i, e := range errors {
		result[i] = fmt.Sprintf("%s: %s", e.Field(), e.Tag())
	}
	return strings.Join(result, ", ")
}

// GetErrorFirstStr returns the first validation error from a validator error.
func GetErrorFirstStr(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return ""
	}

	if len(errors) == 0 {
		return ""
	}

	errArr := errors[0]
	return fmt.Sprintf("%s: %s", errArr.Field(), errArr.Tag())
}
