package provider

import (
"strings"
"testing"
)

func TestChatRequestValidation(t *testing.T) {
tests := []struct {
name    string
req     ChatRequest
wantErr bool
}{
{
name:    "empty model",
req:     ChatRequest{Model: "", Messages: []Message{}},
wantErr: true,
},
{
name:    "empty messages",
req:     ChatRequest{Model: "gpt-4", Messages: []Message{}},
wantErr: true,
},
{
name: "oversized message",
req: ChatRequest{
Model: "gpt-4",
Messages: []Message{
{Role: "user", Content: strings.Repeat("x", 10_000_000)},
},
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := validateChatRequest(&tt.req)
if (err != nil) != tt.wantErr {
t.Errorf("validateChatRequest() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

// validateChatRequest es una función de validación que deberíamos implementar
func validateChatRequest(req *ChatRequest) error {
if req.Model == "" {
return ErrInvalidModel
}
if len(req.Messages) == 0 {
return ErrNoMessages
}
var totalSize int
for _, m := range req.Messages {
totalSize += len(m.Content)
}
if totalSize > 1_000_000 { // 1MB
return ErrRequestTooLarge
}
return nil
}

// Errores de validación
var (
ErrInvalidModel    = &ValidationError{Code: "invalid_model", Message: "model is required"}
ErrNoMessages      = &ValidationError{Code: "no_messages", Message: "at least one message is required"}
ErrRequestTooLarge = &ValidationError{Code: "request_too_large", Message: "request exceeds 1MB limit"}
)

// ValidationError tipo de error de validación
type ValidationError struct {
Code    string
Message string
}

func (e *ValidationError) Error() string {
return e.Code + ": " + e.Message
}
