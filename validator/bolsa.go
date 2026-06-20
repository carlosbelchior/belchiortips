package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type BolsaValidator struct {
	Token      string
	HTTPClient *http.Client
}

func NewBolsaValidator(token string) *BolsaValidator {
	return &BolsaValidator{
		Token: token,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SanitizeTeamName extracts the main part of the team name. Example "Argentina (Tifosi)" -> "Argentina"
func SanitizeTeamName(name string) string {
	parts := strings.Split(name, "(")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(name)
}

// CheckGameExists calls the API and checks if both team names appear in the response body.
func (v *BolsaValidator) CheckGameExists(home, away string) (bool, int, error) {
	url := "https://prod20454-176166000.fssb.io/api/eventlist/eu/leagues/v2/792892689913008128/gameOdds?marketTypeIds=ML39%2CML169%2CML1633%2CML1%2COU249%2COU39%2COU6001%2COU1697%2COU1633%2COU201%2CQA158%2CQA1693%2CML167%2CML0%2CML1%2COU200%2COU201%2CQA158%2CML167&IsLive=true"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0, err
	}

	authHeader := v.Token
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	
	// A API aparentemente exige que o token também seja passado no Cookie
	cookie := os.Getenv("BOLSADEAPOSTA_COOKIE")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	} else {
		req.Header.Set("Cookie", "authorization="+authHeader)
	}

	resp, err := v.HTTPClient.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, 0, fmt.Errorf("status %d - %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}

	var payload struct {
		Data [][]interface{} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return false, 0, fmt.Errorf("erro ao decodificar JSON da Bolsa: %v", err)
	}

	homeSanitized := strings.ToLower(SanitizeTeamName(home))
	awaySanitized := strings.ToLower(SanitizeTeamName(away))

	for _, game := range payload.Data {
		var found bool
		for _, item := range game {
			if str, ok := item.(string); ok {
				if strings.Contains(strings.ToLower(str), homeSanitized) && strings.Contains(strings.ToLower(str), awaySanitized) {
					found = true
					break
				}
			}
		}

		if found {
			totalScore := 0
			// Procurar o array que tenha as pontuações na string ("2", "0")
			for _, item := range game {
				if arr, ok := item.([]interface{}); ok && len(arr) >= 4 {
					s1, ok1 := arr[0].(string)
					s2, ok2 := arr[1].(string)
					// verificar se a quarta posicao é um map/dict (as estatisticas detalhadas)
					_, ok4 := arr[3].(map[string]interface{})
					
					if ok1 && ok2 && ok4 {
						hScore, _ := strconv.Atoi(s1)
						aScore, _ := strconv.Atoi(s2)
						totalScore = hScore + aScore
						break
					}
				}
			}
			return true, totalScore, nil
		}
	}

	return false, 0, nil
}
