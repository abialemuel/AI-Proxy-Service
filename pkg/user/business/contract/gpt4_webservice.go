package contract

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"
)

type GPT4WebService interface {
	Prompt(ctx context.Context, payload gpt4_webservice.GPT4PromptRequestDao) (gpt4_webservice.GPT4PromptResponseDao, error)
}
