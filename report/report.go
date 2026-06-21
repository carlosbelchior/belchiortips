package report

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/carlosbelchior/green365-telegram/green365"
)

var mu sync.Mutex
var reportsDir = "reports"

type ReportEntry struct {
	OpportunityID int64     `json:"opportunityId"`
	Home          string    `json:"home"`
	Away          string    `json:"away"`
	Market        string    `json:"market"`
	TargetLine    string    `json:"targetLine"`
	Odd           float64   `json:"odd"`
	Result        string    `json:"result"` // "win" ou "lose"
	Unit          float64   `json:"unit"`   // Valor ganho/perdido (+0.8, -1.0)
	Timestamp     time.Time `json:"timestamp"`
}

type DailySummary struct {
	Date        string  `json:"date"`
	TotalTips   int     `json:"totalTips"`
	TotalWins   int     `json:"totalWins"`
	TotalLosses int     `json:"totalLosses"`
	TotalProfit float64 `json:"totalProfit"`
}

func init() {
	_ = os.MkdirAll(reportsDir, 0755)
}

func getFileName(date time.Time) string {
	return filepath.Join(reportsDir, fmt.Sprintf("%s.json", date.Format("2006-01-02")))
}

// SaveResult registra uma entrada finalizada no arquivo do dia atual
func SaveResult(opp green365.Opportunity) {
	if opp.ResultStatus == nil || (*opp.ResultStatus != "win" && *opp.ResultStatus != "lose") {
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
	if opp.Unit != nil {
		unit = *opp.Unit
	}
	
	// Tratar valor da unidade
	// O retorno da API para "lose" vem como positivo ou negativo?
	// Geralmente Green365 retorna "lose" com Unit = 1 e a gente precisa garantir que é negativo pro calculo
	if *opp.ResultStatus == "lose" && unit > 0 {
		unit = -unit
	}

	entry := ReportEntry{
		OpportunityID: opp.ID,
		Home:          opp.Home,
		Away:          opp.Away,
		Market:        opp.Market,
		TargetLine:    opp.TargetLine.Name,
		Odd:           opp.TargetLine.Odd,
		Result:        *opp.ResultStatus,
		Unit:          unit,
		Timestamp:     now,
	}

	// Previne duplicação de registro
	for _, e := range entries {
		if e.OpportunityID == entry.OpportunityID {
			return 
		}
	}

	entries = append(entries, entry)
	
	newData, _ := json.MarshalIndent(entries, "", "  ")
	_ = os.WriteFile(filename, newData, 0644)
	
	log.Printf("\033[36mRelatório\033[0m: Entrada %d arquivada como %s", opp.ID, entry.Result)
}

// GetSummary extrai o resumo do dia especificado
func GetSummary(date time.Time) DailySummary {
	mu.Lock()
	defer mu.Unlock()

	filename := getFileName(date)
	summary := DailySummary{
		Date: date.Format("02/01/2006"),
	}

	var entries []ReportEntry
	data, err := os.ReadFile(filename)
	if err != nil {
		return summary 
	}
	_ = json.Unmarshal(data, &entries)

	for _, e := range entries {
		summary.TotalTips++
		if e.Result == "win" {
			summary.TotalWins++
		} else if e.Result == "lose" {
			summary.TotalLosses++
		}
		summary.TotalProfit += e.Unit
	}

	return summary
}
