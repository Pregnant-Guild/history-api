package validator

import (
	"errors"
	"net/url"
	"path"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			name = strings.SplitN(fld.Tag.Get("query"), ",", 2)[0]
		}
		return name
	})

	validate.RegisterValidation("image_url", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		if val == "" {
			return true
		}
		return isImageURL(val)
	})

}

func isImageURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	ext := strings.ToLower(path.Ext(parsed.Path))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

type ErrorResponse struct {
	FailedField string `json:"failed_field"`
	Tag         string `json:"tag"`
	Value       string `json:"value"`
	Message     string `json:"message"`
}

func formatValidationError(err error) []ErrorResponse {
	var validationErrors validator.ValidationErrors
	var errorsList []ErrorResponse

	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			var element ErrorResponse
			element.FailedField = fieldError.Field()
			element.Tag = fieldError.Tag()
			element.Value = fieldError.Param()
			switch fieldError.Tag() {
			case "required":
				element.Message = fieldError.Field() + " is required"
			case "min":
				element.Message = fieldError.Field() + " must be at least " + fieldError.Param() + " characters"
			case "max":
				element.Message = fieldError.Field() + " must be at most " + fieldError.Param() + " characters"
			case "email":
				element.Message = "Invalid email format"
			default:
				element.Message = fieldError.Error()
			}
			errorsList = append(errorsList, element)
		}
	}
	return errorsList
}

func ValidateQueryDto(c fiber.Ctx, dto any) error {
	if err := c.Bind().Query(dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse query parameters: " + err.Error(),
		})
	}

	if err := validate.Struct(dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": formatValidationError(err),
		})
	}

	return nil
}

func ValidateBodyDto(c fiber.Ctx, dto any) error {
	if err := c.Bind().Body(dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	if err := validate.Struct(dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": formatValidationError(err),
		})
	}

	return nil
}
