package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/carlosbelchior/green365-telegram/config"
	"github.com/carlosbelchior/green365-telegram/green365"
	"github.com/carlosbelchior/green365-telegram/report"
	"github.com/carlosbelchior/green365-telegram/telegram"
	"github.com/carlosbelchior/green365-telegram/tracker"
	"github.com/carlosbelchior/green365-telegram/validator"
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

func startDailyReportCron(tBot *telegram.Bot) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			waitDuration := next.Sub(now)
			log.Printf(colorCyan+"[Cron] Relatório diário agendado para rodar em %v"+colorReset, waitDuration.Round(time.Minute))
			
			time.Sleep(waitDuration)

			summary := report.GetSummary(time.Now())
			if summary.TotalTips > 0 {
				msg := fmt.Sprintf("📊 *Fechamento do Dia (%s)*\n\n", summary.Date)
				msg += fmt.Sprintf("🎯 Total de Entradas: %d\n", summary.TotalTips)
				msg += fmt.Sprintf("✅ Greens: %d\n", summary.TotalWins)
				msg += fmt.Sprintf("❌ Reds: %d\n", summary.TotalLosses)
				
				if summary.TotalProfit >= 0 {
					msg += fmt.Sprintf("\n💰 Lucro: +%.2fu 🚀\n", summary.TotalProfit)
				} else {
					msg += fmt.Sprintf("\n📉 Prejuízo: %.2fu ⚠️\n", summary.TotalProfit)
				}
				
				err := tBot.SendTextMessage(msg)
				if err != nil {
					log.Printf(colorRed+"Erro ao enviar relatório diário: %v"+colorReset, err)
				} else {
					log.Println(colorGreen + "Relatório diário enviado com sucesso!" + colorReset)
				}
			}
			
			time.Sleep(time.Minute)
		}
	}()
}

func main() {
	// Custom log format
	log.SetFlags(0)
	log.SetOutput(&customLogWriter{out: os.Stdout})

	// Carrega variaveis do .env se existir, com fallbacks configurados
	config.Load()

	if config.App.Green365Email == "" || config.App.Green365Password == "" || config.App.TelegramBotToken == "" || config.App.TelegramChatID == "" {
		log.Fatal(colorRed + "Variáveis de ambiente incompletas (GREEN365_EMAIL, GREEN365_PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID)" + colorReset)
	}

	botID := config.App.Green365BotID

	log.Println(colorCyan + "Iniciando bot..." + colorReset)

	gClient := green365.NewClient()
	err := gClient.Login(config.App.Green365Email, config.App.Green365Password)
	if err != nil {
		log.Fatalf(colorRed+"Erro ao fazer login no Green365: %v"+colorReset, err)
	}
	log.Println(colorGreen + "Login no Green365 efetuado com sucesso!" + colorReset)

	tBot := telegram.NewBot(config.App.TelegramBotToken, config.App.TelegramChatID)
	trk := tracker.NewTracker()

	startDailyReportCron(tBot)

	bolsaToken := config.App.BolsaDeApostaToken
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
				if err := gClient.Login(config.App.Green365Email, config.App.Green365Password); err != nil {
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

					if opp.Status == "finished" && opp.ResultStatus != nil {
						report.SaveResult(opp)
					}

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
