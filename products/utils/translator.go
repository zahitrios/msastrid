package utils

import (
	"encoding/json"
	"os"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Translation struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type Translator struct {
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
	isLoaded  bool
}

func NewTranslator() *Translator {
	translator := &Translator{}

	if translator.bundle == nil {
		translator.bundle = i18n.NewBundle(language.Spanish)
	}

	if !translator.isLoaded {
		translator.loadMessages()
		translator.isLoaded = true
	}

	if translator.localizer == nil {
		translator.localizer = i18n.NewLocalizer(translator.bundle, language.Spanish.String())

	}

	return translator
}

func Translate(messageId string) string {
	return NewTranslator().message(messageId)
}

func (t *Translator) loadMessages() {
	var translations []Translation

	jsonAsString := []byte(os.Getenv("TRANSLATIONS"))
	json.Unmarshal(jsonAsString, &translations)

	t.bundle = i18n.NewBundle(language.Spanish)

	for _, translation := range translations {
		message := i18n.Message{
			ID:    translation.Id,
			Other: translation.Message,
		}

		t.bundle.AddMessages(language.Spanish, &message)
	}
}

func (t *Translator) message(messageId string) string {
	localizeConfig := i18n.LocalizeConfig{MessageID: messageId}
	localization, _ := t.localizer.Localize(&localizeConfig)

	return localization
}
