package services

import (
	"context"

	"doter/internal/translate"
)

// TranslateService exposes free machine translation to the frontend.
type TranslateService struct {
	client *translate.Client
}

// NewTranslateService creates the service.
func NewTranslateService(client *translate.Client) *TranslateService {
	return &TranslateService{client: client}
}

// Translate converts text into target (a code such as "zh-CN" or "en").
// It returns null when every provider fails; the panel shows that as an error.
func (s *TranslateService) Translate(text, target string) *translate.Result {
	res, err := s.client.Translate(context.Background(), text, target)
	if err != nil {
		return nil
	}
	return res
}
