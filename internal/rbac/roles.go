package rbac

import (
"errors"
)

// Role representa un rol en el sistema
type Role string

const (
RoleAdmin   Role = "admin"
RoleEditor  Role = "editor"
RoleViewer  Role = "viewer"
RoleMember  Role = "member"
)

// Permission representa un permiso
type Permission string

const (
PermRead   Permission = "read"
PermWrite  Permission = "write"
PermDelete Permission = "delete"
PermAdmin  Permission = "admin"
)

var (
ErrOrganizationNotFound = errors.New("organization not found")
ErrProjectNotFound      = errors.New("project not found")
ErrUserAlreadyMember    = errors.New("user is already a member")
ErrUserNotFound         = errors.New("user not found")
ErrInsufficientPermissions = errors.New("insufficient permissions")
ErrInvalidRole          = errors.New("invalid role")
)

// RolePermissions define los permisos por rol
var RolePermissions = map[Role][]Permission{
RoleAdmin:  {PermRead, PermWrite, PermDelete, PermAdmin},
RoleEditor: {PermRead, PermWrite},
RoleViewer: {PermRead},
RoleMember: {PermRead},
}

// HasPermission verifica si un rol tiene un permiso
func HasPermission(role Role, permission Permission) bool {
perms, exists := RolePermissions[role]
if !exists {
return false
}
for _, p := range perms {
if p == permission {
return true
}
}
return false
}

// CheckAccess verifica si un usuario tiene acceso a un recurso
func CheckAccess(role Role, resource string, action Permission) bool {
switch action {
case PermRead:
return HasPermission(role, PermRead)
case PermWrite:
return HasPermission(role, PermWrite)
case PermDelete:
return HasPermission(role, PermDelete)
case PermAdmin:
return HasPermission(role, PermAdmin)
default:
return false
}
}
