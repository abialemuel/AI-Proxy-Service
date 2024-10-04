package gpt4_webservice

// GPT4PromptRequestDao is the request to GPT4 prompt
type GPT4PromptRequestDao struct {
	Model       string       `json:"model"`
	Message     []MessageReq `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens"`
	TopP        float64      `json:"top_p"`
}

// GPT4PromptResponse is the response from GPT4 prompt
type GPT4PromptResponseDao struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created float64   `json:"created"`
	Model   string    `json:"model"`
	Choices []Choices `json:"choices"`
	Usage   Usage     `json:"usage"`
}

type MessageReq struct {
	Content []Content `json:"content"`
	Role    string    `json:"role"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Choices struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
