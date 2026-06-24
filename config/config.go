package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Configuration struct {
	Green365Email       string
	Green365Password    string
	Green365BotIDs      []int
	TelegramBotToken    string
	TelegramChatID      string
	BolsaDeApostaToken  string
	BolsaDeApostaCookie string
	MinOdd              float64
}

var App *Configuration

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvIntSlice(key string, fallback []int) []int {
	if value, ok := os.LookupEnv(key); ok {
		var ints []int
		for _, p := range strings.Split(value, ",") {
			if i, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				ints = append(ints, i)
			}
		}
		if len(ints) > 0 {
			return ints
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

// Load lê o arquivo .env e preenche as configurações com valores padrão (fallbacks)
func Load() {
	// Tenta carregar o .env. Se não achar, não tem problema, usará as vars de ambiente do sistema ou o fallback
	_ = godotenv.Load()

	App = &Configuration{
		// Green365
		Green365Email:    getEnv("GREEN365_EMAIL", "carlosbelchiorsilva@gmail.com"),
		Green365Password: getEnv("GREEN365_PASSWORD", "Cadu@0306"),
		Green365BotIDs:   getEnvIntSlice("BOT_ID", []int{39001}),

		// Telegram
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", "8566402806:AAHU7ARkcY1LSU494y7ZzPoWQ9ernX0b-TY"),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", "-1002043756516"),

		// Bolsa de Aposta
		BolsaDeApostaToken:  getEnv("BOLSADEAPOSTA_TOKEN", ""),
		BolsaDeApostaCookie: getEnv("BOLSADEAPOSTA_COOKIE", ""),

		// Regras de Aposta
		MinOdd: getEnvFloat("MIN_ODD", 1.70),
	}
}
