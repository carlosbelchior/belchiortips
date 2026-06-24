package report

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/carlosbelchior/green365-telegram/config"
	"github.com/carlosbelchior/green365-telegram/green365"
	"github.com/carlosbelchior/green365-telegram/telegram"
)

var mu sync.Mutex
var reportsDir = "reports"

type ReportEntry struct {
	OpportunityID int64     `json:"opportunityId"`
	MessageID     int       `json:"messageId"`
	MarketType    string    `json:"marketType"` // "Ambos Marcam" ou "Over/Under"
	Result        string    `json:"result"`
	Unit          float64   `json:"unit"`
	Odd           float64   `json:"odd"`
	URL           string    `json:"url"` // Para reconstruir o botao do telegram na validacao
	Ignored       bool      `json:"ignored"`
	Timestamp     time.Time `json:"timestamp"`
}

func init() {
	_ = os.MkdirAll(reportsDir, 0755)
}

func getFileName(date time.Time) string {
	return filepath.Join(reportsDir, fmt.Sprintf("%s.json", date.Format("2006-01-02")))
}

func getMarketType(opp green365.Opportunity) string {
	marketLower := strings.ToLower(opp.Market)
	targetLower := strings.ToLower(opp.TargetLine.Name)
	if targetLower == "yes" || targetLower == "no" || strings.Contains(marketLower, "both teams") || strings.Contains(marketLower, "ambos marcam") {
		return "Ambos Marcam"
	}
	return "Over/Under"
}

// SaveResult registra uma entrada finalizada
func SaveResult(opp green365.Opportunity, msgID int) {
	if opp.ResultStatus == nil || (*opp.ResultStatus != "win" && *opp.ResultStatus != "lose" && *opp.ResultStatus != "half_win" && *opp.ResultStatus != "half_lose") {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	filename := getFileName(now)

	var entries []ReportEntry
	data, err := os.ReadFile(filename)
	if err == nil {
		_ = json.Unmarshal(data, &entries)
	}

	unit := 0.0
	if *opp.ResultStatus == "win" {
		unit = config.App.MinOdd - 1.0
	} else if *opp.ResultStatus == "lose" {
		unit = -1.0
	} else if *opp.ResultStatus == "half_win" {
		unit = 0.35
	} else if *opp.ResultStatus == "half_lose" {
		unit = -0.50
	}

	entry := ReportEntry{
		OpportunityID: opp.ID,
		MessageID:     msgID,
		MarketType:    getMarketType(opp),
		Result:        *opp.ResultStatus,
		Unit:          unit,
		Odd:           config.App.MinOdd,
		URL:           opp.TargetLine.SportsBookRedirectURL,
		Ignored:       false,
		Timestamp:     now,
	}

	for i, e := range entries {
		if e.OpportunityID == entry.OpportunityID {
			// Update se já existir
			entries[i] = entry
			newData, _ := json.MarshalIndent(entries, "", "  ")
			_ = os.WriteFile(filename, newData, 0644)
			return
		}
	}

	entries = append(entries, entry)
	newData, _ := json.MarshalIndent(entries, "", "  ")
	_ = os.WriteFile(filename, newData, 0644)
	log.Printf("\033[36mRelatório\033[0m: Entrada %d arquivada como %s", opp.ID, entry.Result)
}

// GenerateDailySummary processa e gera a string de resumo
func GenerateDailySummary(tBot *telegram.Bot) string {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	filename := getFileName(now)

	var entries []ReportEntry
	data, err := os.ReadFile(filename)
	if err != nil {
		return "" // Nenhum relatório hoje
	}
	_ = json.Unmarshal(data, &entries)

	var modified bool
	// Validação de mensagens do telegram
	for i, e := range entries {
		if !e.Ignored && e.MessageID != 0 {
			err := tBot.VerifyMessageExists(e.MessageID, e.URL)
			if err != nil && strings.Contains(strings.ToLower(err.Error()), "not found") {
				entries[i].Ignored = true
				modified = true
				log.Printf("\033[33mRelatório\033[0m: Mensagem %d ignorada (foi deletada do Telegram).", e.MessageID)
			}
		}
	}

	if modified {
		newData, _ := json.MarshalIndent(entries, "", "  ")
		_ = os.WriteFile(filename, newData, 0644)
	}

	var totalTips, totalWins, totalLosses int
	var totalProfit float64
	
	type marketStat struct {
		wins, losses int
		profit float64
	}
	stats := make(map[string]*marketStat)

	for _, e := range entries {
		if e.Ignored {
			continue
		}
		
		if _, ok := stats[e.MarketType]; !ok {
			stats[e.MarketType] = &marketStat{}
		}

		totalTips++
		if e.Result == "win" || e.Result == "half_win" {
			totalWins++
			stats[e.MarketType].wins++
		} else if e.Result == "lose" || e.Result == "half_lose" {
			totalLosses++
			stats[e.MarketType].losses++
		}
		totalProfit += e.Unit
		stats[e.MarketType].profit += e.Unit
	}

	if totalTips == 0 {
		return ""
	}

	msg := fmt.Sprintf("📊 *Fechamento do Dia (%s)*\n\n", now.Format("02/01/2006"))
	msg += fmt.Sprintf("🎯 Total de Entradas: %d\n\n", totalTips)
	msg += fmt.Sprintf("✅ Greens: %d\n\n", totalWins)
	msg += fmt.Sprintf("❌ Reds: %d\n\n", totalLosses)
	
	if totalProfit >= 0 {
		msg += fmt.Sprintf("\n💰 Lucro: +%.2fu 🚀\n\n", totalProfit)
	} else {
		msg += fmt.Sprintf("\n📉 Prejuízo: %.2fu ⚠️\n\n", totalProfit)
	}

	msg += "🔍 *Por Mercado:*\n"
	for mk, st := range stats {
		if st.wins+st.losses > 0 {
			sign := ""
			if st.profit > 0 {
				sign = "+"
			}
			msg += fmt.Sprintf("• %s: %d✅ %d❌ (%s%.2fu)\n\n", mk, st.wins, st.losses, sign, st.profit)
		}
	}

	return msg
}
