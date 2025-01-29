package domain

import (
	"encoding/json"
)

type DataType string

const (
	Object  DataType = "object"
	Number  DataType = "number"
	Integer DataType = "integer"
	String  DataType = "string"
	Array   DataType = "array"
	Null    DataType = "null"
	Boolean DataType = "boolean"
)

type Chat struct {
	ID        int64
	ModelName string
	Messages  []ChatMessage
	Tools     []Tool
}

type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function,omitempty"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  Definition  `json:"parameters"`
	Function    interface{} `json:"-"`
}

type Definition struct {
	Type        DataType              `json:"type,omitempty"`
	Description string                `json:"description,omitempty"`
	Enum        []string              `json:"enum,omitempty"`
	Properties  map[string]Definition `json:"properties"`
	// Required specifies which properties are required, if the schema type is Object.
	Required []string `json:"required,omitempty"`
	// Items specifies which data type an array contains, if the schema type is Array.
	Items *Definition `json:"items,omitempty"`
}

func (d Definition) MarshalJSON() ([]byte, error) {
	if d.Properties == nil {
		d.Properties = make(map[string]Definition)
	}
	type Alias Definition
	return json.Marshal(struct {
		Alias
	}{
		Alias: (Alias)(d),
	})
}
