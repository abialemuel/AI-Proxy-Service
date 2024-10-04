package business

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/config"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business/contract"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/repository"
	"github.com/abialemuel/poly-kit/infrastructure/apm"
	"go.opentelemetry.io/otel/attribute"
)

const (
	userDefaultModel          = "gpt-4o-mini"
	userDefaultTemperature    = 0.7
	userDefaultTop            = 0.95
	userDefaultMaxTokens      = 4096
	userDefaultRole           = "user"
	redisKeyContext           = "context-%s"
	redisKeySummary           = "summary-%s"
	redisKeyTokenUsage        = "token-usage-%s"
	summaryDefaultTemperature = 0.4  // Lower temperature for more focused summaries
	summaryDefaultTop         = 0.65 // Slightly lower for more predictable responses
	summaryDefaultMaxTokens   = 100  // A shorter max token count for concise summaries
	systemBrief               = "Summarize this conversation into key points, keeping essential details without repeating previously given information."
)

// UserService Business Logic of user domain
type UserService struct {
	repo           contract.Repository
	cache          contract.Cache
	cfg            *config.MainConfig
	gpt4Webservice contract.GPT4WebService
}

// NewUserService creates a new instance of UserService
func NewUserService(
	repo contract.Repository,
	cache contract.Cache,
	cfg *config.MainConfig,
	gpt4Webservice contract.GPT4WebService,
) UserService {
	return UserService{repo: repo, cache: cache, cfg: cfg, gpt4Webservice: gpt4Webservice}
}

// UserPromtGPT handles the GPT prompt request for a user
func (u UserService) UserPromtGPT(ctx context.Context, payload core.UserPromtGPTRequest) (res core.UserPromGPTResponse, err error) {
	ctx, span := apm.StartTransaction(ctx, "Service::UserPromtGPT")
	// defer add metadata error to span
	defer func() {
		if err != nil {
			apm.AddEvent(ctx, "Error",
				attribute.String("error", err.Error()),
			)
		}
	}()
	defer apm.EndTransaction(span)

	// Validate token usage
	token, valid, expiredDuration, tokenExist := u.validateTokenUsage(ctx, payload.UserID)
	if !valid {
		return core.UserPromGPTResponse{}, fmt.Errorf("token usage limit reached: %d token, Your limit resets after %s", token, expiredDuration)
	}

	var existingMsgs, existingSummary []gpt4_webservice.MessageReq
	var newSummary gpt4_webservice.MessageReq
	contextKey := fmt.Sprintf(redisKeyContext, payload.UserID)
	summaryKey := fmt.Sprintf(redisKeySummary, payload.UserID)
	newContent := core.ToWebServiceUserPromtGPTContentRequest(payload.Content)

	// Retrieve existing messages from cache
	existingData, success := u.cache.Get(ctx, contextKey)
	if success {
		strData := existingData.(string)
		err = json.Unmarshal([]byte(strData), &existingMsgs)
		if err != nil {
			return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 prompt: Unmarshal: %v", err)
		}
		existingMsgs = append(existingMsgs, gpt4_webservice.MessageReq{
			Content: newContent,
			Role:    userDefaultRole,
		})
	} else {
		existingMsgs = []gpt4_webservice.MessageReq{{
			Content: newContent,
			Role:    userDefaultRole,
		}}
	}

	// Retrieve existing summary from cache
	promptPayload := []gpt4_webservice.MessageReq{{}}
	existingSummaryData, success := u.cache.Get(ctx, summaryKey)
	if success {
		summary := existingSummaryData.(string)
		err = json.Unmarshal([]byte(summary), &existingSummary)
		if err != nil {
			return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 prompt: Unmarshal: %v", err)
		}
		promptPayload = existingSummary
		promptPayload = append(promptPayload, existingMsgs...)
	} else {
		promptPayload = existingMsgs
	}

	// Prepare OpenAI prompt request
	gpt4Payload := gpt4_webservice.GPT4PromptRequestDao{
		Message:     promptPayload,
		Temperature: userDefaultTemperature,
		MaxTokens:   userDefaultMaxTokens,
		TopP:        userDefaultTop,
	}
	gpt4Response, err := u.gpt4Webservice.Prompt(ctx, gpt4Payload)
	if err != nil {
		return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 prompt: %v", err)
	}
	res.GPT4PromptResponse = core.ToCoreGPT4PromptResponse(gpt4Response)
	res.UserID = payload.UserID

	// Append assistant's response to existing messages
	assistantResp := gpt4_webservice.MessageReq{
		Content: []gpt4_webservice.Content{{
			Type: "text",
			Text: &res.GPT4PromptResponse.Choices[0].Message.Content,
		}},
		Role: "assistant",
	}
	existingMsgs = append(existingMsgs, assistantResp)

	// Summarize conversation if message count exceeds threshold
	if len(existingMsgs) >= 10 {
		newSummary, err = u.userSummaryGPT(ctx, existingMsgs, &token)
		if err != nil {
			return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 summary: %v", err)
		}

		existingSummary = append(existingSummary, newSummary)
		jsonData, err := json.Marshal(existingSummary)
		if err != nil {
			return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 summary: %v", err)
		}

		err = u.cache.Set(ctx, summaryKey, jsonData, 0)
		if err != nil {
			return core.UserPromGPTResponse{}, err
		}

		u.cache.Delete(ctx, contextKey)
	} else {
		jsonData, err := json.Marshal(existingMsgs)
		if err != nil {
			return core.UserPromGPTResponse{}, fmt.Errorf("error in GPT4 prompt: %v", err)
		}

		err = u.cache.Set(ctx, contextKey, jsonData, 0)
		if err != nil {
			return core.UserPromGPTResponse{}, err
		}
	}

	// add metadata token usage to span
	apm.AddEvent(ctx, "TokenUsage",
		attribute.String("user_id", payload.UserID),
		attribute.Int("token_usage", res.Usage.TotalTokens),
	)

	// Update token usage
	token += res.Usage.TotalTokens
	duration := time.Second * time.Duration(u.cfg.OpenAI.TokenLifetime)
	if tokenExist {
		duration = -1
		err = u.cache.Set(ctx, fmt.Sprintf(redisKeyTokenUsage, payload.UserID), token, duration)
	} else {
		err = u.cache.Set(ctx, fmt.Sprintf(redisKeyTokenUsage, payload.UserID), token, duration)
	}

	if err != nil {
		return core.UserPromGPTResponse{}, err
	}

	// upsert to mongo
	mongoContent := core.ToContentRepo(payload.Content)
	mongoMessage := []repository.Message{
		{
			Content: mongoContent,
			Role:    userDefaultRole,
		},
		{
			Role: assistantResp.Role,
			Content: []repository.Content{{
				Type: "text",
				Text: assistantResp.Content[0].Text,
			}},
		},
	}

	// newSummary to repo summary if newSummary not nil or empty
	mongoSummary := repository.Summary{}
	if len(newSummary.Content) > 0 {
		mongoSummary = repository.Summary{
			Content: []repository.Content{{
				Type: "text",
				Text: newSummary.Content[0].Text,
			}},
			Role: newSummary.Role,
		}
	}
	go u.upsertConversation(context.Background(), payload.UserID, mongoMessage, mongoSummary)

	return res, nil
}

// ServicePrompt (another backend service)
func (u UserService) ServicePrompt(ctx context.Context, payload core.ServicePromptRequest) (res core.ServicePromGPTResponse, err error) {
	ctx, span := apm.StartTransaction(ctx, "Service::ServicePrompt")
	// defer add metadata error to span
	defer func() {
		if err != nil {
			apm.AddEvent(ctx, "Error",
				attribute.String("error", err.Error()),
			)
		}
	}()
	defer apm.EndTransaction(span)

	// Prepare OpenAI prompt request
	gpt4Payload := gpt4_webservice.GPT4PromptRequestDao{
		Message:     core.ToWebServicePromtGPTMsgRequest(payload.Messages),
		Temperature: payload.Temperature,
		MaxTokens:   payload.MaxTokens,
		TopP:        payload.TopP,
	}
	gpt4Response, err := u.gpt4Webservice.Prompt(ctx, gpt4Payload)
	if err != nil {
		return core.ServicePromGPTResponse{}, fmt.Errorf("error in GPT4 prompt: %v", err)
	}
	res.GPT4PromptResponse = core.ToCoreGPT4PromptResponse(gpt4Response)
	res.UserID = payload.ServiceName

	// add metadata token usage to span
	apm.AddEvent(ctx, "TokenUsage",
		attribute.String("user_id", payload.ServiceName),
		attribute.Int("token_usage", res.Usage.TotalTokens),
	)

	return res, nil
}

// get user token information from cache
func (u UserService) GetUserTokenUsage(ctx context.Context, userID string) core.UserTokenUsage {
	// get token usage from cache
	var tokenCount int
	tokenData, success := u.cache.Get(ctx, fmt.Sprintf(redisKeyTokenUsage, userID))
	if success {
		tokenCount, _ = strconv.Atoi(tokenData.(string))
	} else {
		tokenCount = 0
	}

	warn := false
	if tokenCount > u.cfg.OpenAI.TokenLimit/2 {
		warn = true
	}

	return core.UserTokenUsage{
		TokenLimit: u.cfg.OpenAI.TokenLimit,
		TokenUsage: tokenCount,
		Warning:    warn,
	}
}

// userSummaryGPT generates a summary of the conversation
func (u UserService) userSummaryGPT(ctx context.Context, payload []gpt4_webservice.MessageReq, token *int) (res gpt4_webservice.MessageReq, err error) {
	text := fmt.Sprintf("%s conversation: %s", systemBrief, formatConversation(payload))
	msg := []gpt4_webservice.MessageReq{{
		Content: []gpt4_webservice.Content{{
			Type: "text",
			Text: &text,
		}},
		Role: "system",
	}}
	gpt4Payload := gpt4_webservice.GPT4PromptRequestDao{
		Message:     msg,
		Temperature: summaryDefaultTemperature,
		MaxTokens:   summaryDefaultMaxTokens,
		TopP:        summaryDefaultTop,
	}
	gpt4Response, err := u.gpt4Webservice.Prompt(ctx, gpt4Payload)
	if err != nil {
		return gpt4_webservice.MessageReq{}, err
	}

	*token += gpt4Response.Usage.TotalTokens

	return gpt4_webservice.MessageReq{
		Content: []gpt4_webservice.Content{{
			Type: "text",
			Text: &gpt4Response.Choices[0].Message.Content,
		}},
		Role: "system",
	}, nil
}

// formatConversation formats the conversation for summarization
func formatConversation(conversation []gpt4_webservice.MessageReq) string {
	var formattedText string

	for _, entry := range conversation {
		role := entry.Role
		text := entry.Content[0].Text
		formattedText += role + ": " + *text + "\n"
	}

	return formattedText
}

// validateTokenUsage checks if the user has exceeded their token usage limit
func (u UserService) validateTokenUsage(ctx context.Context, userID string) (token int, valid bool, expired *time.Duration, exist bool) {
	exist = false
	redisKey := fmt.Sprintf(redisKeyTokenUsage, userID)
	var tokenCount int
	tokenData, success := u.cache.Get(ctx, redisKey)
	if success {
		exist = true
		tokenCount, _ = strconv.Atoi(tokenData.(string))
	} else {
		tokenCount = 0
		return tokenCount, true, nil, exist
	}

	if tokenCount > u.cfg.OpenAI.TokenLimit {
		expiredData, _ := u.cache.TTL(ctx, redisKey)
		u.cache.Delete(ctx, fmt.Sprintf(redisKeyContext, userID))
		return tokenCount, false, &expiredData, exist
	}
	return tokenCount, true, nil, exist
}

func (u UserService) upsertConversation(ctx context.Context, userID string, messages []repository.Message, summary repository.Summary) error {
	err := u.repo.UpsertConversation(ctx, userID, messages, &summary)
	if err != nil {
		fmt.Println("Error upserting conversation:", err)
		return err
	}
	return nil
}

// UserClearContext
func (u UserService) UserClearContext(ctx context.Context, userID string) error {
	contextKey := fmt.Sprintf(redisKeyContext, userID)
	summaryKey := fmt.Sprintf(redisKeySummary, userID)

	u.cache.Delete(ctx, contextKey)
	u.cache.Delete(ctx, summaryKey)

	return nil
}
