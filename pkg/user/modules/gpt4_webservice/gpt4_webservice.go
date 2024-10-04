package gpt4_webservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GPT4WebService struct {
	client *http.Client
	url    string
	apiKey string
}

func NewGPT4WebService(url, apiKey string) GPT4WebService {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
		},
	}

	return GPT4WebService{
		client: client,
		url:    url,
		apiKey: apiKey,
	}
}

func (ws GPT4WebService) Prompt(ctx context.Context, payload GPT4PromptRequestDao) (result GPT4PromptResponseDao, err error) {
	jsonBody, _ := json.Marshal(payload)
	reqBody := bytes.NewBuffer(jsonBody)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ws.url, reqBody)
	if err != nil {
		return result, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Api-Key", ws.apiKey)

	response, err := ws.client.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()

	// validate response status code
	if response.StatusCode != http.StatusOK {
		// print response.Body
		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		// print response.Body
		fmt.Println(buf.String())
		// if err this {"error":{"code":"429","message": "Requests to the ChatCompletions_Create Operation under Azure OpenAI API version 2024-02-15-preview have exceeded token rate limit of your current OpenAI S0 pricing tier. Please retry after 8 seconds. Please go here: https://aka.ms/oai/quotaincrease if you would like to further increase the default rate limit."}}, wrap message and give information limitation from Azure OpenAI
		if response.StatusCode == 429 {
			return result, fmt.Errorf("Requests to the ChatCompletions_Create Operation under Azure OpenAI API have exceeded token rate limit of your current OpenAI S0 pricing tier. Please retry later.")
		}
		return result, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}
