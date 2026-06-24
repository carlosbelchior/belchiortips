package green365

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://api-v2.green365.com.br/api/v2"

type Client struct {
	HTTPClient *http.Client
	Token      string
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func (c *Client) Login(email, password string) error {
	reqBody := loginRequest{
		Email:    email,
		Password: password,
	}
	bodyData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL+"/auth/login", bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("falha no login: status %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return err
	}

	c.Token = loginResp.Token
	return nil
}

type TargetLine struct {
	ID                    string  `json:"id"`
	Odd                   float64 `json:"odd"`
	Name                  string  `json:"name"`
	Handicap              float64 `json:"handicap"`
	Suspended             bool    `json:"suspended"`
	SportsBookRedirectURL string  `json:"sportsBookRedirectURL"`
}

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type Opportunity struct {
	ID           int64      `json:"id"`
	BotID        int        `json:"botID"`
	Sport        string     `json:"sport"`
	Status       string     `json:"status"` // "pending", "finished"
	Home         string     `json:"home"`
	Away         string     `json:"away"`
	Competition  string     `json:"competition"`
	CurrentScore Score      `json:"currentScore"`
	FinalScore   Score      `json:"finalScore"`
	MatchResult  *string    `json:"matchResult"`
	ResultStatus *string    `json:"resultStatus"` // "win", "lose"
	Unit         *float64   `json:"unit"`
	Market       string     `json:"market"`
	TargetLine   TargetLine `json:"targetLine"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type getOpportunitiesResponse struct {
	Items []Opportunity `json:"items"`
}

func (c *Client) GetOpportunities(botID int, limit int) ([]Opportunity, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("token não configurado")
	}

	url := fmt.Sprintf("%s/bots/%d/opportunities?limit=%d", baseURL, botID, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("unauthorized") // Needs re-login
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Codigo: %d", resp.StatusCode)
	}

	var getResp getOpportunitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, err
	}

	return getResp.Items, nil
}
