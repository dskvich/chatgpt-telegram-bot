package chatgpt

type imagesGenerationsRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size"`
	ResponseFormat string `json:"response_format"`
}

type imagesGenerationsResponse struct {
	Created int `json:"created"`
	Data    []struct {
		B64Json []byte `json:"b64_json"`
	} `json:"data"`
}
