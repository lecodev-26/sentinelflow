package vault

import (
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"crypto/sha256"
"encoding/base64"
"errors"
"io"
"sync"
"time"
)

// Entry es una entrada de credencial cifrada
type Entry struct {
ID          string    `json:"id"`
Name        string    `json:"name"`
Provider    string    `json:"provider"`
Encrypted   string    `json:"encrypted"` // base64
Nonce       string    `json:"nonce"`     // base64
CreatedAt   time.Time `json:"created_at"`
UpdatedAt   time.Time `json:"updated_at"`
LastAccess  time.Time `json:"last_access,omitempty"`
AccessCount int       `json:"access_count"`
}

// Vault gestiona secretos cifrados
type Vault struct {
mu      sync.RWMutex
key     []byte
entries map[string]*Entry
}

// NewVault crea un vault con una master key
// Si masterKey está vacía, se genera una aleatoria (NO persistente)
func NewVault(masterKey string) (*Vault, error) {
var key []byte
if masterKey == "" {
key = make([]byte, 32)
if _, err := rand.Read(key); err != nil {
return nil, err
}
} else {
// Derivar la key con SHA-256
h := sha256.Sum256([]byte(masterKey))
key = h[:]
}

return &Vault{
key:     key,
entries: make(map[string]*Entry),
}, nil
}

// Store guarda un secreto cifrado
func (v *Vault) Store(id, name, provider, plaintext string) (*Entry, error) {
encrypted, nonce, err := v.encrypt(plaintext)
if err != nil {
return nil, err
}

entry := &Entry{
ID:        id,
Name:      name,
Provider:  provider,
Encrypted: encrypted,
Nonce:     nonce,
CreatedAt: time.Now(),
UpdatedAt: time.Now(),
}

v.mu.Lock()
v.entries[id] = entry
v.mu.Unlock()

return entry, nil
}

// Retrieve obtiene un secreto descifrado
func (v *Vault) Retrieve(id string) (string, error) {
v.mu.RLock()
entry, exists := v.entries[id]
v.mu.RUnlock()

if !exists {
return "", errors.New("entry not found")
}

plaintext, err := v.decrypt(entry.Encrypted, entry.Nonce)
if err != nil {
return "", err
}

v.mu.Lock()
entry.LastAccess = time.Now()
entry.AccessCount++
v.mu.Unlock()

return plaintext, nil
}

// Delete elimina un secreto
func (v *Vault) Delete(id string) bool {
v.mu.Lock()
defer v.mu.Unlock()
if _, exists := v.entries[id]; !exists {
return false
}
delete(v.entries, id)
return true
}

// List lista las entradas (sin descifrar)
func (v *Vault) List() []*Entry {
v.mu.RLock()
defer v.mu.RUnlock()
result := make([]*Entry, 0, len(v.entries))
for _, e := range v.entries {
result = append(result, e)
}
return result
}

// RotateKey rota la master key, re-cifrando todos los secretos
func (v *Vault) RotateKey(newMasterKey string) error {
newKey := sha256.Sum256([]byte(newMasterKey))

// Descifrar con la key antigua
plaintexts := make(map[string]string)
for id, entry := range v.entries {
pt, err := v.decrypt(entry.Encrypted, entry.Nonce)
if err != nil {
return err
}
plaintexts[id] = pt
}

// Cambiar key
v.mu.Lock()
v.key = newKey[:]
v.mu.Unlock()

// Re-cifrar con la nueva
for id, pt := range plaintexts {
encrypted, nonce, err := v.encrypt(pt)
if err != nil {
return err
}
v.mu.Lock()
v.entries[id].Encrypted = encrypted
v.entries[id].Nonce = nonce
v.entries[id].UpdatedAt = time.Now()
v.mu.Unlock()
}

return nil
}

func (v *Vault) encrypt(plaintext string) (string, string, error) {
block, err := aes.NewCipher(v.key)
if err != nil {
return "", "", err
}

gcm, err := cipher.NewGCM(block)
if err != nil {
return "", "", err
}

nonce := make([]byte, gcm.NonceSize())
if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
return "", "", err
}

ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

return base64.StdEncoding.EncodeToString(ciphertext),
base64.StdEncoding.EncodeToString(nonce), nil
}

func (v *Vault) decrypt(encryptedB64, nonceB64 string) (string, error) {
encrypted, err := base64.StdEncoding.DecodeString(encryptedB64)
if err != nil {
return "", err
}

nonce, err := base64.StdEncoding.DecodeString(nonceB64)
if err != nil {
return "", err
}

block, err := aes.NewCipher(v.key)
if err != nil {
return "", err
}

gcm, err := cipher.NewGCM(block)
if err != nil {
return "", err
}

plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
if err != nil {
return "", err
}

return string(plaintext), nil
}
