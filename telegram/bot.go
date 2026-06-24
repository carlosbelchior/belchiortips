package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/carlosbelchior/green365-telegram/config"
	"github.com/carlosbelchior/green365-telegram/green365"
)

const telegramAPI = "https://api.telegram.org/bot%s"

type Bot struct {
	Token      string
	ChatID     string
	HTTPClient *http.Client
}

func NewBot(token, chatID string) *Bot {
	return &Bot{
		Token:  token,
		ChatID: chatID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type sendMessageRequest struct {
	ChatID      string               `json:"chat_id"`
	Text        string               `json:"text"`
	ParseMode   string               `json:"parse_mode"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type editMessageRequest struct {
	ChatID      string               `json:"chat_id"`
	MessageID   int                  `json:"message_id"`
	Text        string               `json:"text"`
	ParseMode   string               `json:"parse_mode"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type telegramResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
	Description string `json:"description"`
}

func getFlag(teamName string) string {
	baseName := teamName
	if idx := strings.Index(teamName, " ("); idx != -1 {
		baseName = teamName[:idx]
	}
	// Normalizar minúsculas
	baseName = strings.TrimSpace(strings.ToLower(baseName))

	if flag, ok := countryFlags[baseName]; ok {
		return flag + " " + teamName
	}
	return teamName
}

func FormatMessage(opp green365.Opportunity) string {
	text := fmt.Sprintf("🔵 Ao vivo\n\n")
	
	targetName := opp.TargetLine.Name
	// Tradução para Ambos Marcam
	if strings.ToLower(targetName) == "yes" {
		targetName = "Ambos Marcam SIM"
	} else if strings.ToLower(targetName) == "no" {
		targetName = "Ambos Marcam NÃO"
	}
	
	text += fmt.Sprintf("📈 *%s - @%.2f*\n\n", targetName, config.App.MinOdd)
	
	homeStr := getFlag(opp.Home)
	awayStr := getFlag(opp.Away)
	
	text += fmt.Sprintf("🏃 *%s* vs *%s*\n\n", homeStr, awayStr)

	if opp.Status == "finished" && opp.ResultStatus != nil {
		if *opp.ResultStatus == "win" {
			profit := config.App.MinOdd - 1.0
			text += fmt.Sprintf("🏆 Resultado: +%.2fu ✅\n\n", profit)
		} else if *opp.ResultStatus == "lose" {
			loss := -1.0
			text += fmt.Sprintf("❌ Resultado: %.2fu\n\n", loss)
		} else if *opp.ResultStatus == "half_win" {
			text += fmt.Sprintf("🏆 Resultado: +0.35u (Meio Green) ✅\n\n")
		} else if *opp.ResultStatus == "half_lose" {
			text += fmt.Sprintf("❌ Resultado: -0.50u (Meio Red)\n\n")
		}
	}

	text += "JOGUE COM RESPONSABILIDADE +18"
	return text
}

func (b *Bot) getReplyMarkup(url string) *InlineKeyboardMarkup {
	if url == "" {
		return nil
	}
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔗 Ir para o jogo", URL: url},
			},
		},
	}
}

func (b *Bot) SendMessage(opp green365.Opportunity) (int, error) {
	url := fmt.Sprintf(telegramAPI+"/sendMessage", b.Token)

	reqBody := sendMessageRequest{
		ChatID:      b.ChatID,
		Text:        FormatMessage(opp),
		ParseMode:   "Markdown",
		ReplyMarkup: b.getReplyMarkup(opp.TargetLine.SportsBookRedirectURL),
	}

	bodyData, _ := json.Marshal(reqBody)
	resp, err := b.HTTPClient.Post(url, "application/json", bytes.NewBuffer(bodyData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var tResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return 0, err
	}

	if !tResp.Ok {
		return 0, fmt.Errorf("erro no telegram: %s", tResp.Description)
	}

	return tResp.Result.MessageID, nil
}

func (b *Bot) EditMessage(messageID int, opp green365.Opportunity) error {
	url := fmt.Sprintf(telegramAPI+"/editMessageText", b.Token)

	reqBody := editMessageRequest{
		ChatID:      b.ChatID,
		MessageID:   messageID,
		Text:        FormatMessage(opp),
		ParseMode:   "Markdown",
		ReplyMarkup: b.getReplyMarkup(opp.TargetLine.SportsBookRedirectURL),
	}

	bodyData, _ := json.Marshal(reqBody)
	resp, err := b.HTTPClient.Post(url, "application/json", bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return err
	}

	if !tResp.Ok {
		return fmt.Errorf("erro ao editar telegram: %s", tResp.Description)
	}

	return nil
}

func (b *Bot) SendTextMessage(text string) error {
	url := fmt.Sprintf(telegramAPI+"/sendMessage", b.Token)

	reqBody := sendMessageRequest{
		ChatID:    b.ChatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	bodyData, _ := json.Marshal(reqBody)
	resp, err := b.HTTPClient.Post(url, "application/json", bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return err
	}

	if !tResp.Ok {
		return fmt.Errorf("erro no telegram: %s", tResp.Description)
	}

	return nil
}

type editMarkupRequest struct {
	ChatID      string               `json:"chat_id"`
	MessageID   int                  `json:"message_id"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// VerifyMessageExists attempts to edit the reply markup to the same value.
// If the message is deleted, Telegram returns a "message to edit not found" error.
// If it exists, it returns "message is not modified" (or succeeds).
func (b *Bot) VerifyMessageExists(messageID int, url string) error {
	apiURL := fmt.Sprintf(telegramAPI+"/editMessageReplyMarkup", b.Token)

	reqBody := editMarkupRequest{
		ChatID:      b.ChatID,
		MessageID:   messageID,
		ReplyMarkup: b.getReplyMarkup(url),
	}

	bodyData, _ := json.Marshal(reqBody)
	resp, err := b.HTTPClient.Post(apiURL, "application/json", bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return err
	}

	if !tResp.Ok {
		return fmt.Errorf("erro no telegram: %s", tResp.Description)
	}

	return nil
}
