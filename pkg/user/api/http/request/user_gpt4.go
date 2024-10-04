package request

import "github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"

//	   "content": [
//	                {
//	                    "type": "text",
//	                    "text": "What’s in this image?",
//	                    "image_url": {}
//	                },
//	                {
//	                    "type": "image_url",
//	                    "image_url": {
//	                        "url": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/4gHYS"
//						}}
type UserPromptGPTRequest struct {
	Content []Content `json:"content" validate:"required"`
}

type Content struct {
	Type     string    `json:"type" validate:"required"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

func ToCoreUserPromptGPTRequest(req []Content) (res []core.Content) {
	for _, v := range req {
		var content core.Content
		content.Type = v.Type
		if v.Text != nil {
			content.Text = v.Text
		}
		if v.ImageURL != nil {
			content.ImageURL = &core.ImageURL{
				URL: v.ImageURL.URL,
			}
		}
		res = append(res, content)
	}
	return res

}
