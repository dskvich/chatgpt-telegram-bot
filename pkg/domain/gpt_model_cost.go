package domain

type GptModelCost struct {
	Context   float64
	Generated float64
	Image     float64
}

func DefaultGptModelCosts() map[string]GptModelCost {
	return map[string]GptModelCost{
		"gpt-3.5-turbo-0301":        {Context: 0.0015, Generated: 0.002},
		"gpt-3.5-turbo-0613":        {Context: 0.0015, Generated: 0.002},
		"gpt-3.5-turbo-16k":         {Context: 0.003, Generated: 0.004},
		"gpt-3.5-turbo-16k-0613":    {Context: 0.003, Generated: 0.004},
		"gpt-4-0314":                {Context: 0.03, Generated: 0.06},
		"gpt-4-0613":                {Context: 0.03, Generated: 0.06},
		"gpt-4-32k":                 {Context: 0.06, Generated: 0.12},
		"gpt-4-32k-0314":            {Context: 0.06, Generated: 0.12},
		"gpt-4-32k-0613":            {Context: 0.06, Generated: 0.12},
		"text-embedding-ada-002-v2": {Context: 0.0001, Generated: 0},
		"whisper-1":                 {Context: 0.006 / 60.0, Generated: 0},
		"dalle-512x512":             {Image: 0.018},
		"dalle-1024x1024":           {Image: 0.020},
		"dalle-256x256":             {Image: 0.016},
	}
}
