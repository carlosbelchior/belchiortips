package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/carlosbelchior/green365-telegram/green365"
	"github.com/carlosbelchior/green365-telegram/telegram"
	"github.com/carlosbelchior/green365-telegram/tracker"
	"github.com/joho/godotenv"
)

func getCombinedStatus(opp green365.Opportunity) string {
	res := "nil"
	if opp.ResultStatus != nil {
		res = *opp.ResultStatus
	}
	return fmt.Sprintf("%s|%s", opp.Status, res)
}

func main() {
	// Carrega variaveis do .env se existir
	_ = godotenv.Load()

	email := os.Getenv("GREEN365_EMAIL")
	password := os.Getenv("GREEN365_PASSWORD")
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	
	if email == "" || password == "" || telegramToken == "" || telegramChatID == "" {
		log.Fatal("Variáveis de ambiente incompletas (GREEN365_EMAIL, GREEN365_PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID)")
	}

	botIDStr := os.Getenv("BOT_ID")
	botID := 38974
	if botIDStr != "" {
		if id, err := strconv.Atoi(botIDStr); err == nil {
			botID = id
		}
	}

	log.Println("Iniciando bot...")

	gClient := green365.NewClient()
	err := gClient.Login(email, password)
	if err != nil {
		log.Fatalf("Erro ao fazer login no Green365: %v", err)
	}
	log.Println("Login no Green365 efetuado com sucesso!")

	tBot := telegram.NewBot(telegramToken, telegramChatID)
	trk := tracker.NewTracker()

	for {
		log.Println("Buscando novas entradas...")
		opps, err := gClient.GetOpportunities(botID, 10)
		if err != nil {
			if err.Error() == "unauthorized" {
				log.Println("Token expirado, refazendo login...")
				if err := gClient.Login(email, password); err != nil {
					log.Printf("Erro ao refazer login: %v", err)
				}
			} else {
				log.Printf("Erro ao buscar oportunidades: %v", err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		for _, opp := range opps {
			msgID, exists := trk.GetMessageID(opp.ID)
			currentCombinedStatus := getCombinedStatus(opp)

			if !exists {
				// Nova oportunidade
				if opp.Status == "pending" {
					if time.Since(opp.CreatedAt) > 8*time.Minute {
						log.Printf("Ignorando entrada (ID: %d): criada há %v (> 8min)", opp.ID, time.Since(opp.CreatedAt).Round(time.Second))
						trk.SaveMessageID(opp.ID, 0, currentCombinedStatus)
						continue
					}
					log.Printf("Nova entrada pendente detectada (ID: %d): %s vs %s", opp.ID, opp.Home, opp.Away)
					newMsgID, err := tBot.SendMessage(opp)
					if err != nil {
						log.Printf("Erro ao enviar mensagem para Telegram: %v", err)
						trk.SaveMessageID(opp.ID, 0, currentCombinedStatus)
					} else {
						trk.SaveMessageID(opp.ID, newMsgID, currentCombinedStatus)
					}
				}
			} else {
				// Oportunidade existente
				lastCombinedStatus, _ := trk.GetStatus(opp.ID)
				
				if currentCombinedStatus != lastCombinedStatus {
					// Status mudou
					log.Printf("Atualizando entrada (ID: %d): mudou de %s para %s", opp.ID, lastCombinedStatus, currentCombinedStatus)
					if msgID != 0 {
						err := tBot.EditMessage(msgID, opp)
						if err != nil {
							log.Printf("Erro ao editar mensagem no Telegram: %v", err)
						} else {
							trk.UpdateStatus(opp.ID, currentCombinedStatus)
						}
					} else {
						trk.UpdateStatus(opp.ID, currentCombinedStatus)
					}
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}
