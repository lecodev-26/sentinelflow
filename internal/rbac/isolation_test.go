package rbac

import (
"testing"
"time"
)

func TestTenantIsolation(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

orgA := orgMgr.CreateOrganization("org-a", "Organization A")
orgB := orgMgr.CreateOrganization("org-b", "Organization B")

userA, err := userMgr.CreateUser("a@a.com", "User A", RoleAdmin, orgA.ID)
if err != nil {
t.Fatalf("Error creating user A: %v", err)
}
userB, err := userMgr.CreateUser("b@b.com", "User B", RoleAdmin, orgB.ID)
if err != nil {
t.Fatalf("Error creating user B: %v", err)
}

keyA, _, err := userMgr.CreateAPIKey(userA.ID, "key-a", "", nil, 365*24*time.Hour)
if err != nil {
t.Fatalf("Error creating key A: %v", err)
}
keyB, _, err := userMgr.CreateAPIKey(userB.ID, "key-b", "", nil, 365*24*time.Hour)
if err != nil {
t.Fatalf("Error creating key B: %v", err)
}

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

if apiKeyA.OrgID == apiKeyB.OrgID {
t.Error("Tenant isolation broken")
}
}

func TestKeyHashNotExposed(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

org := orgMgr.CreateOrganization("test", "Test")
user, _ := userMgr.CreateUser("test@test.com", "Test", RoleAdmin, org.ID)
rawKey, apiKey, _ := userMgr.CreateAPIKey(user.ID, "test-key", "", nil, 365*24*time.Hour)

if rawKey == apiKey.KeyHash {
t.Error("Raw key should not equal hash")
}
if len(apiKey.KeyHash) != 64 {
t.Errorf("KeyHash should be 64 chars, got %d", len(apiKey.KeyHash))
}
if len(rawKey) < 3 || rawKey[:3] != "sf_" {
t.Error("Raw key should start with sf_")
}
}

func TestRevokedKeyRejected(t *testing.T) {
orgMgr := NewOrganizationManager()
userMgr := NewUserManager(orgMgr)

org := orgMgr.CreateOrganization("test", "Test")
user, _ := userMgr.CreateUser("test@test.com", "Test", RoleAdmin, org.ID)
rawKey, _, _ := userMgr.CreateAPIKey(user.ID, "test-key", "", nil, 365*24*time.Hour)

_, _, valid := userMgr.ValidateAPIKey(rawKey)
if !valid {
t.Fatal("Key should be valid initially")
}

userMgr.RevokeAPIKey(rawKey)

_, _, valid = userMgr.ValidateAPIKey(rawKey)
if valid {
t.Error("Revoked key should not validate")
}
}
