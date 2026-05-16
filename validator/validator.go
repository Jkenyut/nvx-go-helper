// Package validator provides an interface wrapper for go-playground/validator/v10
// to simplify struct validation and support thread-safe custom error message translation.
package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Validator defines the standard interface for structural and variable validation.
// It is recommended to use this interface in your services for easy dependency injection and mocking.
type Validator interface {
	// Struct validates a struct and returns the first error encountered, or nil.
	Struct(s any) error

	// Var validates a single variable against a tag.
	Var(field any, tag string) error

	// RegisterCustomValidation registers a custom validation tag and its human-readable error message.
	RegisterCustomValidation(tag string, fn validator.Func, customMsg string) error

	// GetErrors returns the raw validator.ValidationErrors.
	GetErrors(err error) validator.ValidationErrors

	// GetErrorsFullStr returns the full validation errors in a single comma-separated string.
	GetErrorsFullStr(err error) string

	// GetErrorFirstStr returns the first validation error as a string.
	GetErrorFirstStr(err error) string

	// GetErrorFirstMsg returns the first validation error translated into a user-friendly message.
	GetErrorFirstMsg(err error) string

	// GetErrorsFullMsg returns all validation errors translated into user-friendly messages.
	GetErrorsFullMsg(err error) string

	// Raw returns the underlying *v10.Validate instance for advanced usage.
	Raw() *validator.Validate
}

// wrapper implements the Validator interface.
type wrapper struct {
	v               *validator.Validate
	customMsgMap    map[string]string
	customMsgMapMux sync.RWMutex
}

// New creates and returns a new thread-safe Validator instance.
func New() Validator {
	v := validator.New()

	// RegisterTagNameFunc registers a function to get the tag name from the struct field.
	// This is used to return the JSON tag name in validation errors.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &wrapper{
		v:            v,
		customMsgMap: make(map[string]string),
	}
}

// Raw returns the underlying *v10.Validate instance.
func (w *wrapper) Raw() *validator.Validate {
	return w.v
}

// Struct validates a struct.
func (w *wrapper) Struct(s any) error {
	return w.v.Struct(s)
}

// Var validates a single variable.
func (w *wrapper) Var(field any, tag string) error {
	return w.v.Var(field, tag)
}

// RegisterCustomValidation registers a custom validation tag and its human-readable error message.
func (w *wrapper) RegisterCustomValidation(tag string, fn validator.Func, customMsg string) error {
	if err := w.v.RegisterValidation(tag, fn); err != nil {
		return err
	}

	w.customMsgMapMux.Lock()
	w.customMsgMap[tag] = customMsg
	w.customMsgMapMux.Unlock()

	return nil
}

// GetErrors returns the validation errors from a validator error.
func (w *wrapper) GetErrors(err error) validator.ValidationErrors {
	if err == nil {
		return nil
	}
	if errs, ok := err.(validator.ValidationErrors); ok {
		return errs
	}
	return nil
}

// GetErrorsFullStr returns the full validation errors.
func (w *wrapper) GetErrorsFullStr(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
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

// GetErrorFirstStr returns the first validation error.
func (w *wrapper) GetErrorFirstStr(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	if len(errors) == 0 {
		return ""
	}

	errArr := errors[0]
	return fmt.Sprintf("%s: %s", errArr.Field(), errArr.Tag())
}

// msgForTag translates standard validator tags into user-friendly messages.
func (w *wrapper) msgForTag(fe validator.FieldError) string {
	// 1. Check Custom Messages Registry
	w.customMsgMapMux.RLock()
	customMsg, exists := w.customMsgMap[fe.Tag()]
	w.customMsgMapMux.RUnlock()

	if exists {
		if strings.Contains(customMsg, "%s") {
			return fmt.Sprintf(customMsg, fe.Param())
		}
		return customMsg
	}

	// 2. Fallback to Standard Messages
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address format"
	case "min":
		return fmt.Sprintf("Minimum length or value is %s", fe.Param())
	case "max":
		return fmt.Sprintf("Maximum length or value is %s", fe.Param())
	case "len":
		return fmt.Sprintf("Length must be exactly %s", fe.Param())
	case "gte":
		return fmt.Sprintf("Must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("Must be less than or equal to %s", fe.Param())
	case "gt":
		return fmt.Sprintf("Must be greater than %s", fe.Param())
	case "lt":
		return fmt.Sprintf("Must be less than %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of [%s]", strings.ReplaceAll(fe.Param(), " ", ", "))
	case "url":
		return "Invalid URL format"
	case "uuid":
		return "Invalid UUID format"
	case "alphanum":
		return "Can only contain alphanumeric characters"
	case "alpha":
		return "Can only contain alphabetic characters"
	case "numeric":
		return "Must be a valid numeric value"
	case "number":
		return "Must be a valid number"
	case "boolean":
		return "Must be a boolean value"
	case "contains":
		return fmt.Sprintf("Must contain the text '%s'", fe.Param())
	case "containsany":
		return fmt.Sprintf("Must contain at least one of the following characters: '%s'", fe.Param())
	case "excludes":
		return fmt.Sprintf("Must not contain the text '%s'", fe.Param())
	case "excludesall":
		return fmt.Sprintf("Must not contain any of the following characters: '%s'", fe.Param())
	case "startswith":
		return fmt.Sprintf("Must start with '%s'", fe.Param())
	case "endswith":
		return fmt.Sprintf("Must end with '%s'", fe.Param())
	case "eq":
		return fmt.Sprintf("Must be equal to %s", fe.Param())
	case "ne":
		return fmt.Sprintf("Must not be equal to %s", fe.Param())
	case "eqfield":
		return fmt.Sprintf("Must match %s field", fe.Param())
	case "nefield":
		return fmt.Sprintf("Must not match %s field", fe.Param())
	case "ip", "ipv4", "ipv6":
		return "Must be a valid IP address"
	case "mac":
		return "Must be a valid MAC address"
	case "base64":
		return "Must be a valid Base64 string"
	case "datetime":
		return fmt.Sprintf("Must be a valid datetime format (%s)", fe.Param())
	case "json":
		return "Must be a valid JSON string"
	default:
		return fmt.Sprintf("Failed on %s validation", fe.Tag())
	}
}

// GetErrorFirstMsg returns the first validation error translated into a user-friendly message.
func (w *wrapper) GetErrorFirstMsg(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error() // Fallback to raw error if not a ValidationErrors
	}

	if len(errors) == 0 {
		return ""
	}

	errArr := errors[0]
	return fmt.Sprintf("%s: %s", errArr.Field(), w.msgForTag(errArr))
}

// GetErrorsFullMsg returns all validation errors translated into user-friendly messages.
func (w *wrapper) GetErrorsFullMsg(err error) string {
	if err == nil {
		return ""
	}

	var errors validator.ValidationErrors
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	if len(errors) == 0 {
		return ""
	}

	var result = make([]string, len(errors))
	for i, e := range errors {
		result[i] = fmt.Sprintf("%s: %s", e.Field(), w.msgForTag(e))
	}
	return strings.Join(result, ", ")
}

// ============================================================================
// Global Singleton (Backward Compatibility)
// ============================================================================

var (
	defaultValidator Validator
	once             sync.Once
)

// Get returns the global default Validator singleton.
// It creates a single Validator instance thread-safely.
func Get() Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

// Struct validates a struct using the global validator.
func Struct(s any) error {
	return Get().Struct(s)
}

// Var validates a single variable using the global validator.
func Var(field any, tag string) error {
	return Get().Var(field, tag)
}

// RegisterCustomValidation registers a custom validation tag to the global validator.
func RegisterCustomValidation(tag string, fn validator.Func, customMsg string) error {
	return Get().RegisterCustomValidation(tag, fn, customMsg)
}

// GetErrors returns the validation errors from a validator error using the global validator.
func GetErrors(err error) validator.ValidationErrors {
	return Get().GetErrors(err)
}

// GetErrorsFullStr returns the full validation errors using the global validator.
func GetErrorsFullStr(err error) string {
	return Get().GetErrorsFullStr(err)
}

// GetErrorFirstStr returns the first validation error using the global validator.
func GetErrorFirstStr(err error) string {
	return Get().GetErrorFirstStr(err)
}

// GetErrorFirstMsg returns the first translated validation error using the global validator.
func GetErrorFirstMsg(err error) string {
	return Get().GetErrorFirstMsg(err)
}

// GetErrorsFullMsg returns all translated validation errors using the global validator.
func GetErrorsFullMsg(err error) string {
	return Get().GetErrorsFullMsg(err)
}
