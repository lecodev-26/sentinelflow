package gateway

import (
"encoding/json"
"net/http"
)

// WriteError escribe un error normalizado al cliente
func WriteError(w http.ResponseWriter, err error) {
gwErr := AsGatewayError(err)
if gwErr == nil {
gwErr = NewInternalError("unknown error", nil)
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(gwErr.StatusCode)
json.NewEncoder(w).Encode(map[string]interface{}{
"error": map[string]interface{}{
"type":    gwErr.Type,
"message": gwErr.Message,
"details": gwErr.Details,
},
})
}

// WriteJSON escribe una respuesta JSON exitosa
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(data)
}
