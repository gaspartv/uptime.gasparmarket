package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Env struct {
	Port                    string   `validate:"required"`
	APIChecksURL            []string `validate:"required,min=1"`
	APIWhatsAppFakeURL      string   `validate:"required"`
	APIWhatsAppFakeInstance string   `validate:"required"`
	APIWhatsAppFakeToken    string   `validate:"required"`
	WhatsAppSenderNumbers   []string `validate:"required,min=1"`
}

func ValidateEnv() (*Env, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	var env Env
	env.Port = os.Getenv("PORT")
	env.APIChecksURL = parseChecksURLs(os.Getenv("API_CHECKS_URL"))
	env.APIWhatsAppFakeURL = strings.TrimSpace(os.Getenv("API_WHATSAPP_FAKE_URL"))
	env.APIWhatsAppFakeInstance = strings.TrimSpace(os.Getenv("API_WHATSAPP_FAKE_INSTANCE"))
	env.APIWhatsAppFakeToken = strings.TrimSpace(os.Getenv("API_WHATSAPP_FAKE_TOKEN"))
	env.WhatsAppSenderNumbers = parseSenderNumbers(os.Getenv("WHATSAPP_SENDER_NUMBERS"))

	if len(env.APIChecksURL) == 0 {
		return nil, fmt.Errorf("API_CHECKS_URL is required and must contain at least one valid URL")
	}

	validate := validator.New()
	if err := validate.Struct(env); err != nil {
		return nil, err
	}

	return &env, nil
}

func parseChecksURLs(rawURLs string) []string {
	rawURLs = strings.TrimSpace(rawURLs)
	if rawURLs == "" {
		return nil
	}

	var urls []string
	if err := json.Unmarshal([]byte(rawURLs), &urls); err != nil {
		return nil
	}

	cleaned := make([]string, 0, len(urls))
	for _, candidate := range urls {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			cleaned = append(cleaned, candidate)
		}
	}

	return cleaned
}

func parseSenderNumbers(rawNumbers string) []string {
	parts := strings.Split(rawNumbers, ",")
	numbers := make([]string, 0, len(parts))

	for _, part := range parts {
		candidate := strings.TrimSpace(strings.Trim(part, "\""))
		if candidate == "" {
			continue
		}

		numbers = append(numbers, candidate)
	}

	return numbers
}
