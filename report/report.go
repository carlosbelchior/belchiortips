package report

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/carlosbelchior/green365-telegram/config"
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

type MonthlySummary struct {
	Month         string  `json:"month"`
	TotalTips     int     `json:"totalTips"`
	TotalWins     int     `json:"totalWins"`
	TotalLosses   int     `json:"totalLosses"`
	TotalProfit   float64 `json:"totalProfit"`
	MaxWinStreak  int     `json:"maxWinStreak"`
	MaxLoseStreak int     `json:"maxLoseStreak"`
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
	if *opp.ResultStatus == "win" {
		unit = config.App.MinOdd - 1.0
	} else if *opp.ResultStatus == "lose" {
		unit = -1.0
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

// GetMonthlySummary lê todos os relatórios do mês especificado e calcula o fechamento com sequências
func GetMonthlySummary(year int, month time.Month) MonthlySummary {
	mu.Lock()
	defer mu.Unlock()

	summary := MonthlySummary{
		Month: fmt.Sprintf("%02d/%d", month, year),
	}

	var allEntries []ReportEntry

	// Loop por todos os dias do mês
	for day := 1; day <= 31; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if date.Month() != month {
			break // Passou do último dia do mês
		}

		filename := getFileName(date)
		data, err := os.ReadFile(filename)
		if err == nil {
			var dailyEntries []ReportEntry
			if json.Unmarshal(data, &dailyEntries) == nil {
				allEntries = append(allEntries, dailyEntries...)
			}
		}
	}

	currentWinStreak := 0
	currentLoseStreak := 0

	for _, e := range allEntries {
		summary.TotalTips++
		if e.Result == "win" {
			summary.TotalWins++
			currentWinStreak++
			currentLoseStreak = 0
			if currentWinStreak > summary.MaxWinStreak {
				summary.MaxWinStreak = currentWinStreak
			}
		} else if e.Result == "lose" {
			summary.TotalLosses++
			currentLoseStreak++
			currentWinStreak = 0
			if currentLoseStreak > summary.MaxLoseStreak {
				summary.MaxLoseStreak = currentLoseStreak
			}
		}
		summary.TotalProfit += e.Unit
	}

	return summary
}
