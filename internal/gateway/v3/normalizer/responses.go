package normalizer

import (
	"encoding/json"
	"fmt"
)

type ResponsesRequest struct {
	Model        string      `json:"model"`
	Input        interface{} `json:"input"`
	Instructions string      `json:"instructions,omitempty"`
	Stream       bool        `json:"stream,omitempty"`
	Tools        []Tool      `json:"tools,omitempty"`
}

func (n *Normalizer) NormalizeResponses(raw []byte) (*Request, error) {
	var in ResponsesRequest
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid responses JSON: %w", err)
	}
	if in.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	var msgs []Message
	if in.Instructions != "" {
		msgs = append(msgs, Message{Role: "system", Content: in.Instructions})
	}
	switch v := in.Input.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("input is required")
		}
		msgs = append(msgs, Message{Role: "user", Content: v})
	case []interface{}:
		for _, item := range v {
			b, _ := json.Marshal(item)
			var m Message
			if json.Unmarshal(b, &m) == nil && m.Role != "" && m.Content != "" {
				msgs = append(msgs, m)
				continue
			}
			var text string
			if obj, ok := item.(map[string]interface{}); ok {
				if s, ok := obj["content"].(string); ok {
					text = s
				}
			}
			if text != "" {
				msgs = append(msgs, Message{Role: "user", Content: text})
			}
		}
	default:
		return nil, fmt.Errorf("input must be a string or message array")
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("input is required")
	}
	return &Request{Model: in.Model, Messages: msgs, Stream: in.Stream, Tools: in.Tools}, nil
}
