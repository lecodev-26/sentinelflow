package provider

import (
"encoding/json"
"testing"
)

// FuzzChatRequest prueba el parser de ChatRequest con inputs aleatorios
func FuzzChatRequest(f *testing.F) {
// Seeds
seeds := []string{
`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`,
`{"model":"","messages":[]}`,
`{}`,
`{"model":"gpt-4","messages":null}`,
`{"model":"gpt-4","messages":[{"role":"","content":""}]}`,
`{"model":"gpt-4","temperature":0.5,"max_tokens":100}`,
`{"stream":true}`,
`{"model":"a","messages":[{"role":"user","content":"x"}]}`,
`{"model":"a","messages":[{"role":"user"}]}`,
`{"model":"a","messages":[{}]}`,
}
for _, s := range seeds {
f.Add([]byte(s))
}

f.Fuzz(func(t *testing.T, data []byte) {
var req ChatRequest
err := json.Unmarshal(data, &req)
if err != nil {
// Es válido que falle con datos malformados
return
}

// Invariantes que siempre deben cumplirse
_ = req.Model
_ = req.Messages
_ = req.Stream
// No debe hacer panic
})
}

// FuzzMessage prueba el parser de Message
func FuzzMessage(f *testing.F) {
seeds := []string{
`{"role":"user","content":"hi"}`,
`{"role":"system","content":"sys"}`,
`{"role":"assistant","content":"ok"}`,
`{}`,
`{"role":null,"content":null}`,
}
for _, s := range seeds {
f.Add([]byte(s))
}

f.Fuzz(func(t *testing.T, data []byte) {
var msg Message
_ = json.Unmarshal(data, &msg)
// No debe hacer panic
})
}
