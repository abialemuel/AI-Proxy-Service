package validator

import (
	"fmt"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
)

var (
	validate *validator.Validate
	uni      *ut.UniversalTranslator
	trans    ut.Translator
)

//GetValidator Initiatilize validator in singleton way
func GetValidator() *validator.Validate {

	if validate == nil {
		validate = validator.New()
	}

	return validate
}

func getTrans() ut.Translator {
	if trans == nil {
		en := en.New()
		uni = ut.New(en, en)
		trans, _ = uni.GetTranslator("en")

		enTranslations.RegisterDefaultTranslations(validate, trans)
	}

	return trans
}

func CreateValidationErrorMessage(err error) string {
	for _, e := range err.(validator.ValidationErrors) {
		translatedErr := fmt.Errorf(e.Translate(getTrans()))
		return translatedErr.Error()
	}
	return err.Error()
}

func Validation(reg interface{}) (string, bool) {
	var message string
	var check bool = true

	err := GetValidator().Struct(reg)
	if err != nil {
		_, check = err.(*echo.HTTPError)
		if !check {
			message = CreateValidationErrorMessage(err)
		}
	}
	return message, check
}
