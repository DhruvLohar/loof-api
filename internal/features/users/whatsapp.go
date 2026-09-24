package users

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"loof/internal/config"
)

type WhatsAppPayload struct {
	MessagingProduct string           `json:"messaging_product"`
	To               string           `json:"to"`
	Type             string           `json:"type"`
	Template         WhatsAppTemplate `json:"template"`
}

type WhatsAppTemplate struct {
	Name       string              `json:"name"`
	Language   WhatsAppLanguage    `json:"language"`
	Components []WhatsAppInterface `json:"components"`
}

type WhatsAppLanguage struct {
	Code string `json:"code"`
}

// WhatsAppInterface can represent both body and button components
type WhatsAppInterface struct {
	Type       string              `json:"type"`
	SubType    string              `json:"sub_type,omitempty"`
	Index      string              `json:"index,omitempty"`
	Parameters []WhatsAppParameter `json:"parameters"`
}

type WhatsAppParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// CleanPhoneNumber removes '+' and any non-digit characters
func CleanPhoneNumber(countryCode, phoneNumber string) string {
	var sb strings.Builder
	for _, r := range countryCode + phoneNumber {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GenerateOTP generates a cryptographically secure 6-digit OTP
func GenerateOTP() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return 0, err
	}
	return int(n.Int64() + 100000), nil
}

// SendWhatsAppOTP sends a 6-digit OTP using the WhatsApp Cloud API
func SendWhatsAppOTP(ctx context.Context, countryCode, phoneNumber string, otp int) error {
	accessToken := config.GetEnv("WHATSAPP_ACCESS_TOKEN")
	phoneNumberID := config.GetEnv("WHATSAPP_PHONE_NUMBER_ID")
	templateName := config.GetEnv("WHATSAPP_TEMPLATE_NAME")

	if accessToken == "" || phoneNumberID == "" || templateName == "" {
		return fmt.Errorf("missing WhatsApp configuration: ACCESS_TOKEN=%t, PHONE_NUMBER_ID=%t, TEMPLATE_NAME=%t",
			accessToken != "", phoneNumberID != "", templateName != "")
	}

	cleanedTo := CleanPhoneNumber(countryCode, phoneNumber)
	if cleanedTo == "" {
		return fmt.Errorf("invalid destination phone number: countryCode=%s, phoneNumber=%s", countryCode, phoneNumber)
	}

	payload := WhatsAppPayload{
		MessagingProduct: "whatsapp",
		To:               cleanedTo,
		Type:             "template",
		Template: WhatsAppTemplate{
			Name: templateName,
			Language: WhatsAppLanguage{
				Code: "en",
			},
			Components: []WhatsAppInterface{
				{
					Type: "body",
					Parameters: []WhatsAppParameter{
						{
							Type: "text",
							Text: fmt.Sprintf("%06d", otp),
						},
					},
				},
				{
					Type:    "button",
					SubType: "url",
					Index:   "0",
					Parameters: []WhatsAppParameter{
						{
							Type: "text",
							Text: fmt.Sprintf("%06d", otp),
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp payload: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("whatsapp API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
