package main

import (
	"errors"
	"fmt"

	"github.com/audryus/valid8"
	enlocale "github.com/go-playground/locales/en"
	ptlocale "github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	pt_translations "github.com/go-playground/validator/v10/translations/pt_BR"
)

type Lero struct {
	Leros []Lero2 `json:"leros" validate:"dive"`
}

type Lero2 struct {
	Letra string `json:"letra" validate:"alphaspace"`
}

func main() {
	validate := validator.New()

	l := Lero{
		Leros: []Lero2{
			{Letra: "a 2c%"},
		},
	}

	err := validate.Struct(l)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, e := range validationErrors {
			fmt.Println(e.Error())
			fmt.Println(e.Namespace())
			fmt.Println(e.Field())
			fmt.Println(e.StructNamespace())
			fmt.Println(e.StructField())
			fmt.Println(e.Tag())
			fmt.Println(e.ActualTag())
			fmt.Println(e.Kind())
			fmt.Println(e.Type())
			fmt.Println(e.Value())
			fmt.Println(e.Param())
			fmt.Println()
		}
	}

	// setup translator
	en := enlocale.New()
	pt := ptlocale.New()

	uni := ut.New(en, pt)

	transEN, _ := uni.GetTranslator("en")
	transPT, _ := uni.GetTranslator("pt_BR")
	transES, _ := uni.GetTranslator("es")
	en_translations.RegisterDefaultTranslations(validate, transEN)
	pt_translations.RegisterDefaultTranslations(validate, transPT)

	/*validate.RegisterTranslation("alphanum", transEN,
		func(ut ut.Translator) error {
			return ut.Add("alphanum", "{0} must asda only alphanumeric characters", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("alphanum", fe.Field())
			return t
		},
	)*/

	err = validate.Struct(l)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			fmt.Println(e.Translate(transPT))
			fmt.Println(e.Translate(transEN))
			fmt.Println(e.Translate(transES))
		}
	}

	vv := valid8.New(valid8.WithLocales(valid8.DE))

	err = vv.RegisterTranslation(valid8.Translation{
		Tag:    "alphaspace",
		Text:   "{0} ist nicht aa gültig",
		Locale: valid8.DE,
	})

	erros := vv.Struct(l, valid8.DE)

	mapa := valid8.ErrorsToMap(erros)
	fmt.Printf("%+v\n", mapa)

}
