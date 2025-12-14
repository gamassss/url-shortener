package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gamassss/url-shortener/pkg/response"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

var reservedKeywords = map[string]bool{
	"api": true,
}

func init() {
	validate = validator.New()

	validate.RegisterValidation("alias", validateAlias)
}

func Validate(data interface{}) []response.ValidationError {
	var validationErrors []response.ValidationError

	err := validate.Struct(data)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, response.ValidationError{
				Field:   err.Field(),
				Message: getErrorMessage(err),
			})
		}
	}

	return validationErrors
}

func validateAlias(fl validator.FieldLevel) bool {
	alias := fl.Field().String()

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, alias)
	return matched
}

func IsReservedKeyword(alias string) bool {
	return reservedKeywords[strings.ToLower(alias)]
}

func getErrorMessage(err validator.FieldError) string {
	field := err.Field()

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, err.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, err.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, err.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
