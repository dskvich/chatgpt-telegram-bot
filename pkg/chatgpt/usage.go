package chatgpt

type gptDataType struct {
	AggregationTimestamp  int    `json:"aggregation_timestamp"`
	NRequests             int    `json:"n_requests"`
	Operation             string `json:"operation"`
	SnapshotId            string `json:"snapshot_id"`
	NContext              int    `json:"n_context"`
	NContextTokensTotal   int    `json:"n_context_tokens_total"`
	NGenerated            int    `json:"n_generated"`
	NGeneratedTokensTotal int    `json:"n_generated_tokens_total"`
}

type gptWhisperApiDataType struct {
	Timestamp   int    `json:"timestamp"`
	ModelId     string `json:"model_id"`
	NumSeconds  int    `json:"num_seconds"`
	NumRequests int    `json:"num_requests"`
}

type gptDalleApiDataType struct {
	Timestamp   int    `json:"timestamp"`
	NumImages   int    `json:"num_images"`
	NumRequests int    `json:"num_requests"`
	ImageSize   string `json:"image_size"`
	Operation   string `json:"operation"`
}

type gptUsageData struct {
	Object          string                  `json:"object"`
	Data            []gptDataType           `json:"data"`
	FtData          []interface{}           `json:"ft_data"`
	DalleApiData    []gptDalleApiDataType   `json:"dalle_api_data"`
	WhisperApiData  []gptWhisperApiDataType `json:"whisper_api_data"`
	CurrentUsageUsd float64                 `json:"current_usage_usd"`
}
