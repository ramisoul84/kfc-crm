package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validatorOnce     sync.Once
	validatorInstance *Validator

	phoneRegex = regexp.MustCompile(`^(\+7|8)\d{10}$`)
)

// Validator wraps go-playground/validator with custom rules.
type Validator struct {
	validate *validator.Validate
}

// ValidationError describes a single field failure.
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationErrors is a slice of field failures.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return ""
	}
	msgs := make([]string, len(ve.Errors))
	for i, e := range ve.Errors {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// New creates (or returns) the singleton Validator.
func New() *Validator {
	validatorOnce.Do(func() {
		v := validator.New()

		// Use JSON tag names in error messages.
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "" || name == "-" {
				return fld.Name
			}
			return name
		})

		// Register custom validators.
		_ = v.RegisterValidation("phone", validatePhone)
		_ = v.RegisterValidation("uppercase", validateUppercase)
		_ = v.RegisterValidation("role", validateRole)

		validatorInstance = &Validator{validate: v}
	})
	return validatorInstance
}

// Validate validates a struct.
func (v *Validator) Validate(s interface{}) error {
	if s == nil {
		return ValidationErrors{
			Errors: []ValidationError{
				{Field: "struct", Message: "cannot validate nil", Tag: "nil"},
			},
		}
	}

	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		return fmt.Errorf("invalid validation error: %w", err)
	}

	fieldErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	out := ValidationErrors{Errors: make([]ValidationError, 0, len(fieldErrors))}
	for _, fe := range fieldErrors {
		out.Errors = append(out.Errors, ValidationError{
			Field:   fe.Field(),
			Message: messageFor(fe),
			Tag:     fe.Tag(),
			Value:   fe.Value(),
		})
	}
	return out
}

// ValidateVar validates a single value against a tag.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	err := v.validate.Var(field, tag)
	if err == nil {
		return nil
	}

	fieldErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	out := ValidationErrors{Errors: make([]ValidationError, 0, len(fieldErrors))}
	for _, fe := range fieldErrors {
		out.Errors = append(out.Errors, ValidationError{
			Field:   fe.Field(),
			Message: messageFor(fe),
			Tag:     fe.Tag(),
			Value:   fe.Value(),
		})
	}
	return out
}

// AsValidationErrors extracts ValidationErrors from an error.
func AsValidationErrors(err error) (ValidationErrors, bool) {
	ve, ok := err.(ValidationErrors)
	return ve, ok
}

// ─────────────────────────────────────────────────────────────────
// Custom validators
// ─────────────────────────────────────────────────────────────────

func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true
	}
	phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(phone)
	return phoneRegex.MatchString(phone)
}

func validateUppercase(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return s == strings.ToUpper(s)
}

func validateRole(fl validator.FieldLevel) bool {
	return isValidRole(fl.Field().String())
}

// isValidRole is duplicated from domain to avoid a dependency cycle
// between pkg and internal.
func isValidRole(role string) bool {
	switch role {
	case "super_admin", "regional_manager", "restaurant_manager", "shift_manager", "cashier":
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────
// Message formatting
// ─────────────────────────────────────────────────────────────────

func messageFor(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, fe.Param())
	case "phone":
		return fmt.Sprintf("%s must be a valid phone number", field)
	case "uppercase":
		return fmt.Sprintf("%s must be uppercase", field)
	case "role":
		return fmt.Sprintf("%s must be a valid role", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// Ensure the uuid import is used (kept for future UUID-based rules).
var _ = uuid.Nil
