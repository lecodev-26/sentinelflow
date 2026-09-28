package v3

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}
type ToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type ToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionTool `json:"function"`
}
type FunctionTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}
