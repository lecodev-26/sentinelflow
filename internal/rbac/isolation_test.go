package rbac

import (
"testing"
)

func TestTenantIsolation(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

// Crear dos organizaciones
orgA := orgMgr.CreateOrganization("org-a", "Organization A")
orgB := orgMgr.CreateOrganization("org-b", "Organization B")

// Crear usuarios
userA, err := userMgr.CreateUser("a@a.com", "User A", RoleAdmin, orgA.ID)
if err != nil {
t.Fatalf("Error creating user A: %v", err)
}
userB, err := userMgr.CreateUser("b@b.com", "User B", RoleAdmin, orgB.ID)
if err != nil {
t.Fatalf("Error creating user B: %v", err)
}

// Crear API keys
keyA, _, err := userMgr.CreateAPIKey(userA.ID, "key-a", "")
if err != nil {
t.Fatalf("Error creating key A: %v", err)
}
keyB, _, err := userMgr.CreateAPIKey(userB.ID, "key-b", "")
if err != nil {
t.Fatalf("Error creating key B: %v", err)
}

// Validar key A → debe pertenecer a org A
apiKeyA, userA2, validA := userMgr.ValidateAPIKey(keyA)
if !validA {
t.Fatal("Key A should be valid")
}
if apiKeyA.OrgID != orgA.ID {
t.Errorf("Key A should belong to org A, got %s", apiKeyA.OrgID)
}
if userA2.OrgID != orgA.ID {
t.Errorf("User A should belong to org A")
}

// Validar key B → debe pertenecer a org B
apiKeyB, userB2, validB := userMgr.ValidateAPIKey(keyB)
if !validB {
t.Fatal("Key B should be valid")
}
if apiKeyB.OrgID != orgB.ID {
t.Errorf("Key B should belong to org B, got %s", apiKeyB.OrgID)
}
if userB2.OrgID != orgB.ID {
t.Errorf("User B should belong to org B")
}

// Verificar que son tenants distintos
if apiKeyA.OrgID == apiKeyB.OrgID {
t.Error("Tenant isolation broken: keys from different orgs share OrgID")
}
}

func TestKeyHashNotExposed(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

org := orgMgr.CreateOrganization("test", "Test")
user, _ := userMgr.CreateUser("test@test.com", "Test", RoleAdmin, org.ID)
rawKey, apiKey, _ := userMgr.CreateAPIKey(user.ID, "test-key", "")

// El raw key NO debe ser igual al hash
if rawKey == apiKey.KeyHash {
t.Error("Raw key should not equal hash")
}

// El hash debe tener 64 caracteres (SHA-256 hex)
if len(apiKey.KeyHash) != 64 {
t.Errorf("KeyHash should be 64 chars, got %d", len(apiKey.KeyHash))
}

// La key raw debe empezar con sf_
if len(rawKey) < 3 || rawKey[:3] != "sf_" {
t.Error("Raw key should start with sf_")
}
}

func TestRevokedKeyRejected(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

org := orgMgr.CreateOrganization("test", "Test")
user, _ := userMgr.CreateUser("test@test.com", "Test", RoleAdmin, org.ID)
rawKey, _, _ := userMgr.CreateAPIKey(user.ID, "test-key", "")

// Debe validar
_, _, valid := userMgr.ValidateAPIKey(rawKey)
if !valid {
t.Fatal("Key should be valid initially")
}

// Revocar
if !userMgr.RevokeAPIKey(rawKey) {
t.Fatal("Revoke should succeed")
}

// Ya no debe validar
_, _, valid = userMgr.ValidateAPIKey(rawKey)
if valid {
t.Error("Revoked key should not validate")
}
}
