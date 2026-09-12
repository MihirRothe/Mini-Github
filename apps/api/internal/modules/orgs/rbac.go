package orgs

// PermissionPriority returns the numeric hierarchy level of a permission.
func PermissionPriority(p Permission) int {
	switch p {
	case PermAdmin:
		return 5
	case PermMaintain:
		return 4
	case PermWrite:
		return 3
	case PermTriage:
		return 2
	case PermRead:
		return 1
	default:
		return 0
	}
}

// Includes checks if permission p includes or exceeds the required target permission.
func (p Permission) Includes(target Permission) bool {
	return PermissionPriority(p) >= PermissionPriority(target)
}

// MaxPermission returns the higher of two permissions.
func MaxPermission(p1, p2 Permission) Permission {
	if PermissionPriority(p1) >= PermissionPriority(p2) {
		return p1
	}
	return p2
}

// OrgRoleToDefaultRepoPerm maps an organization role to default repository permissions.
func OrgRoleToDefaultRepoPerm(role OrgRole) Permission {
	switch role {
	case OrgRoleOwner:
		return PermAdmin
	case OrgRoleAdmin:
		return PermMaintain
	case OrgRoleMember:
		return PermNone
	default:
		return PermNone
	}
}

// ResolveEffectivePermission calculates the effective permission on a repository
// by combining site administration, org-level role, direct repo membership, and team permissions.
func ResolveEffectivePermission(
	isSiteAdmin bool,
	orgRole *OrgRole,
	directRepoPerm *Permission,
	teamPerms []Permission,
) Permission {
	// 1. Site administrators always have full admin privileges
	if isSiteAdmin {
		return PermAdmin
	}

	effective := PermNone

	// 2. Organization owners automatically have full admin permissions
	if orgRole != nil {
		if *orgRole == OrgRoleOwner {
			return PermAdmin
		}
		effective = MaxPermission(effective, OrgRoleToDefaultRepoPerm(*orgRole))
	}

	// 3. Direct collaborator permissions
	if directRepoPerm != nil {
		effective = MaxPermission(effective, *directRepoPerm)
	}

	// 4. Team permissions (user gets highest permission across all their teams)
	for _, tp := range teamPerms {
		effective = MaxPermission(effective, tp)
	}

	return effective
}
