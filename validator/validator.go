// Package validator provides a singleton instance of go-playground/validator
// and wrapper functions to simplify struct validation.
//
// It ensures consistent validation rules across the application.
package validator

import (
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
		// You can register custom validation tags here in the future
		// e.g., validate.RegisterValidation("sku", validateSKU)
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
