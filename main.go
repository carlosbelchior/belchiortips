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
	"github.com/carlosbelchior/green365-telegram/validator"
	"github.com/joho/godotenv"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

type customLogWriter struct {
	out *os.File
}

func (c *customLogWriter) Write(p []byte) (n int, err error) {
	now := time.Now().Format("02/01 15:04 ")
	res := append([]byte(now), p...)
	return c.out.Write(res)
}

func getCombinedStatus(opp green365.Opportunity) string {
	res := "nil"
	if opp.ResultStatus != nil {
		res = *opp.ResultStatus
	}
	return fmt.Sprintf("%s|%s", opp.Status, res)
}

func processNewTip(opp green365.Opportunity, targetStatus string, tBot *telegram.Bot, trk *tracker.Tracker, bValidator *validator.BolsaValidator) {
	if bValidator == nil {
		sendTip(opp, targetStatus, tBot, trk)
		return
	}

	// Loop por 10 minutos (20 vezes de 30 segundos)
	for i := 0; i < 20; i++ {
		exists, _, err := bValidator.CheckGameExists(opp.Home, opp.Away)
		if err != nil {
			log.Printf(colorRed+"Erro ao consultar Bolsa de Aposta (ID: %d): %v"+colorReset, opp.ID, err)
		} else if exists {
			log.Println(colorGreen + "Partida encontrada! Enviando para o telegram..." + colorReset)
			sendTip(opp, targetStatus, tBot, trk)
			return
		}
		
		time.Sleep(30 * time.Second)
	}

	log.Println(colorYellow + "Tempo esgotado. Jogo não encontrado no Bolsa de Aposta..." + colorReset)
	trk.UpdateStatus(opp.ID, targetStatus)
}

func sendTip(opp green365.Opportunity, targetStatus string, tBot *telegram.Bot, trk *tracker.Tracker) {
	newMsgID, err := tBot.SendMessage(opp)
	if err != nil {
		log.Printf(colorRed+"Erro ao enviar mensagem para Telegram: %v"+colorReset, err)
		trk.UpdateStatus(opp.ID, targetStatus) 
	} else {
		trk.SaveMessageID(opp.ID, newMsgID, targetStatus)
	}
}

func main() {
	// Custom log format
	log.SetFlags(0)
	log.SetOutput(&customLogWriter{out: os.Stdout})

	// Carrega variaveis do .env se existir
	_ = godotenv.Load()

	email := os.Getenv("GREEN365_EMAIL")
	password := os.Getenv("GREEN365_PASSWORD")
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	
	if email == "" || password == "" || telegramToken == "" || telegramChatID == "" {
		log.Fatal(colorRed + "Variáveis de ambiente incompletas (GREEN365_EMAIL, GREEN365_PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID)" + colorReset)
	}

	botIDStr := os.Getenv("BOT_ID")
	botID := 38974
	if botIDStr != "" {
		if id, err := strconv.Atoi(botIDStr); err == nil {
			botID = id
		}
	}

	log.Println(colorCyan + "Iniciando bot..." + colorReset)

	gClient := green365.NewClient()
	err := gClient.Login(email, password)
	if err != nil {
		log.Fatalf(colorRed+"Erro ao fazer login no Green365: %v"+colorReset, err)
	}
	log.Println(colorGreen + "Login no Green365 efetuado com sucesso!" + colorReset)

	tBot := telegram.NewBot(telegramToken, telegramChatID)
	trk := tracker.NewTracker()

	bolsaToken := os.Getenv("BOLSADEAPOSTA_TOKEN")
	var bValidator *validator.BolsaValidator
	if bolsaToken != "" {
		bValidator = validator.NewBolsaValidator(bolsaToken)
		log.Println(colorGreen + "Validador da Bolsa de Aposta ativado!" + colorReset)
	} else {
		log.Println(colorYellow + "AVISO: BOLSADEAPOSTA_TOKEN não configurado. As apostas serão enviadas sem validação." + colorReset)
	}

	for {
		log.Println("Buscando novas entradas...")
		opps, err := gClient.GetOpportunities(botID, 10)
		if err != nil {
			if err.Error() == "unauthorized" {
				log.Println(colorYellow + "Token expirado, refazendo login..." + colorReset)
				if err := gClient.Login(email, password); err != nil {
					log.Printf(colorRed+"Erro ao refazer login: %v"+colorReset, err)
				}
			} else {
				log.Printf(colorRed+"Erro ao buscar oportunidades: %v"+colorReset, err)
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
						log.Printf(colorYellow+"Entrada %d expirada..."+colorReset, opp.ID)
						trk.SaveMessageID(opp.ID, 0, currentCombinedStatus)
						continue
					}
					
					// Marca como validando no tracker provisoriamente para nao criar multiplas threads
					trk.SaveMessageID(opp.ID, 0, "validating")
					
					log.Printf(colorCyan+"Validacao em background para entrada %d: %s vs %s..."+colorReset, opp.ID, opp.Home, opp.Away)
					go processNewTip(opp, currentCombinedStatus, tBot, trk, bValidator)
				}
			} else {
				// Oportunidade existente
				lastCombinedStatus, _ := trk.GetStatus(opp.ID)
				
				if currentCombinedStatus != lastCombinedStatus {
					// Status mudou
					log.Printf(colorCyan+"Atualizando entrada %d"+colorReset, opp.ID)
					if msgID != 0 {
						err := tBot.EditMessage(msgID, opp)
						if err != nil {
							log.Println(colorYellow + "Mensagem do Telegram não encontrada. Ignorando..." + colorReset)
						}
						trk.UpdateStatus(opp.ID, currentCombinedStatus)
					} else {
						trk.UpdateStatus(opp.ID, currentCombinedStatus)
					}
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}
