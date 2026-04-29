package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gaspartv/uptime.gasparmarket/src/config"
)

type CheckResult struct {
	URL        string        `json:"url"`
	Online     bool          `json:"online"`
	StatusCode int           `json:"statusCode,omitempty"`
	Error      string        `json:"error,omitempty"`
	Latency    time.Duration `json:"latency"`
	CheckedAt  time.Time     `json:"checkedAt"`
}

type UptimeService struct {
	httpClient *http.Client
	env        *config.Env
}

func NewUptimeService(timeout time.Duration, env *config.Env) *UptimeService {
	return &UptimeService{
		httpClient: &http.Client{Timeout: timeout},
		env:        env,
	}
}

func (s *UptimeService) Start(ctx context.Context, urls []string, onlineInterval time.Duration, offlineInterval time.Duration) {
	if len(urls) == 0 {
		return
	}

	if onlineInterval <= 0 {
		onlineInterval = time.Minute
	}

	if offlineInterval <= 0 {
		offlineInterval = 5 * time.Minute
	}

	for _, targetURL := range urls {
		go s.monitorTarget(ctx, targetURL, onlineInterval, offlineInterval)
	}

	go s.startHeartbeat(ctx)
}

func (s *UptimeService) monitorTarget(ctx context.Context, targetURL string, onlineInterval time.Duration, offlineInterval time.Duration) {
	nextInterval := onlineInterval
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			result := s.checkTarget(ctx, targetURL)
			if result.Online {
				nextInterval = onlineInterval
			} else {
				log.Printf("[uptime] %s is offline: %s", result.URL, result.Error)

				if err := s.notifyWhatsApp(ctx, result); err != nil {
					log.Printf("[uptime] failed to notify WhatsApp for %s: %v", result.URL, err)
				}

				nextInterval = offlineInterval
			}

			timer.Reset(nextInterval)
		}
	}
}

func (s *UptimeService) CheckNow(ctx context.Context, urls []string) []CheckResult {
	results := make([]CheckResult, 0, len(urls))
	for _, targetURL := range urls {
		results = append(results, s.checkTarget(ctx, targetURL))
	}

	return results
}

func (s *UptimeService) runCheck(ctx context.Context, urls []string) {
	results := s.CheckNow(ctx, urls)
	for _, result := range results {
		if result.Online {
			continue
		}

		log.Printf("[uptime] %s is offline: %s", result.URL, result.Error)

		if err := s.notifyWhatsApp(ctx, result); err != nil {
			log.Printf("[uptime] failed to notify WhatsApp for %s: %v", result.URL, err)
		}
	}
}

func (s *UptimeService) checkTarget(ctx context.Context, targetURL string) CheckResult {
	startedAt := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return CheckResult{
			URL:       targetURL,
			Online:    false,
			Error:     err.Error(),
			Latency:   time.Since(startedAt),
			CheckedAt: startedAt,
		}
	}

	response, err := s.httpClient.Do(request)
	if err != nil {
		return CheckResult{
			URL:       targetURL,
			Online:    false,
			Error:     err.Error(),
			Latency:   time.Since(startedAt),
			CheckedAt: startedAt,
		}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	online := response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
	result := CheckResult{
		URL:        targetURL,
		Online:     online,
		StatusCode: response.StatusCode,
		Latency:    time.Since(startedAt),
		CheckedAt:  startedAt,
	}

	if !online {
		result.Error = http.StatusText(response.StatusCode)
		if result.Error == "" {
			result.Error = "target returned a non-success status code"
		}
	}

	return result
}

func (s *UptimeService) notifyWhatsApp(ctx context.Context, result CheckResult) error {
	message := buildOfflineMessage(result)
	whatsappURL := strings.TrimRight(s.env.APIWhatsAppFakeURL, "/") + "/message/sendText/" + url.PathEscape(s.env.APIWhatsAppFakeInstance)
	var sendErrors []error

	for _, number := range s.env.WhatsAppSenderNumbers {
		if err := s.sendWhatsApp(ctx, whatsappURL, number, message); err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("send whatsapp to %s: %w", number, err))
		}
	}

	if len(sendErrors) > 0 {
		return errors.Join(sendErrors...)
	}

	return nil
}

func (s *UptimeService) sendWhatsApp(ctx context.Context, whatsappURL string, number string, message string) error {
	payload := map[string]any{
		"number": number,
		"options": map[string]any{
			"delay": 1200,
		},
		"textMessage": map[string]any{
			"text": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, whatsappURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.env.APIWhatsAppFakeToken)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if len(responseBody) > 0 {
			return fmt.Errorf("whatsapp api returned status %s for number %s: %s", response.Status, number, strings.TrimSpace(string(responseBody)))
		}

		return fmt.Errorf("whatsapp api returned status %s for number %s", response.Status, number)
	}

	return nil
}

func buildOfflineMessage(result CheckResult) string {
	return fmt.Sprintf("Alerta de uptime: %s está offline. Status: %d. Erro: %s", result.URL, result.StatusCode, result.Error)
}

var heartbeatMessages = []string{
	"Serviço de mensagem e de uptime funcionando corretamente %d",
	"Sistema de notificação está operacional %d",
	"Uptime monitor e WhatsApp em perfeito funcionamento %d",
	"Verificação de saúde dos serviços: tudo ok %d",
	"Sistema respondendo normalmente %d",
	"Monitor de disponibilidade ativo e funcionando %d",
	"Serviços de notificação e uptime estão online %d",
	"Heartbeat: tudo operacional neste momento %d",
}

func (s *UptimeService) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendHeartbeat(ctx)
		}
	}
}

func (s *UptimeService) sendHeartbeat(ctx context.Context) {
	randNum := rand.Intn(1000000)
	messageTemplate := heartbeatMessages[rand.Intn(len(heartbeatMessages))]
	message := fmt.Sprintf(messageTemplate, randNum)

	whatsappURL := strings.TrimRight(s.env.APIWhatsAppFakeURL, "/") + "/message/sendText/" + url.PathEscape(s.env.APIWhatsAppFakeInstance)

	for _, number := range s.env.WhatsAppSenderNumbers {
		if err := s.sendWhatsApp(ctx, whatsappURL, number, message); err != nil {
			log.Printf("[uptime] failed to send heartbeat to %s: %v", number, err)
		} else {
			log.Printf("[uptime] heartbeat sent to %s: %s", number, message)
		}
	}
}
