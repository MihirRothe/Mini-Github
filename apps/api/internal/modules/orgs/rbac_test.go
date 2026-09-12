package orgs

import (
	"context"
	"testing"

	"forgehub/apps/api/internal/modules/auth"
)

func TestPermissionHierarchy(t *testing.T) {
	if !PermAdmin.Includes(PermMaintain) {
		t.Errorf("admin should include maintain")
	}
	if !PermAdmin.Includes(PermWrite) {
		t.Errorf("admin should include write")
	}
	if !PermAdmin.Includes(PermRead) {
		t.Errorf("admin should include read")
	}
	if !PermWrite.Includes(PermRead) {
		t.Errorf("write should include read")
	}
	if PermRead.Includes(PermWrite) {
		t.Errorf("read should NOT include write")
	}
	if PermTriage.Includes(PermWrite) {
		t.Errorf("triage should NOT include write")
	}
	if !PermWrite.Includes(PermTriage) {
		t.Errorf("write should include triage")
	}
}

func TestResolveEffectivePermission(t *testing.T) {
	ownerRole := OrgRoleOwner
	adminRole := OrgRoleAdmin
	memberRole := OrgRoleMember

	writePerm := PermWrite
	readPerm := PermRead

	// 1. Site Admin has PermAdmin regardless of anything else
	p := ResolveEffectivePermission(true, &memberRole, &readPerm, nil)
	if p != PermAdmin {
		t.Errorf("expected PermAdmin for site admin, got %s", p)
	}

	// 2. Org Owner has PermAdmin
	p = ResolveEffectivePermission(false, &ownerRole, nil, nil)
	if p != PermAdmin {
		t.Errorf("expected PermAdmin for org owner, got %s", p)
	}

	// 3. Org Admin has PermMaintain by default
	p = ResolveEffectivePermission(false, &adminRole, nil, nil)
	if p != PermMaintain {
		t.Errorf("expected PermMaintain for org admin, got %s", p)
	}

	// 4. Org Member has PermNone by default
	p = ResolveEffectivePermission(false, &memberRole, nil, nil)
	if p != PermNone {
		t.Errorf("expected PermNone for org member with no assignments, got %s", p)
	}

	// 5. Org Member with Direct Write Collaborator permission gets PermWrite
	p = ResolveEffectivePermission(false, &memberRole, &writePerm, nil)
	if p != PermWrite {
		t.Errorf("expected PermWrite for direct write collaborator, got %s", p)
	}

	// 6. Org Member with Team Write permission gets PermWrite
	p = ResolveEffectivePermission(false, &memberRole, nil, []Permission{PermRead, PermWrite})
	if p != PermWrite {
		t.Errorf("expected PermWrite from highest team perm, got %s", p)
	}

	// 7. Highest permission wins: direct read vs team write -> write
	p = ResolveEffectivePermission(false, &memberRole, &readPerm, []Permission{PermWrite})
	if p != PermWrite {
		t.Errorf("expected PermWrite when team write exceeds direct read, got %s", p)
	}
}

func TestOrgAndTeamWorkflows(t *testing.T) {
	ctx := context.Background()
	authRepo := auth.NewMemoryRepository()

	// Seed users
	u1 := &auth.User{Username: "alice", Email: "alice@example.com", DisplayName: "Alice"}
	u2 := &auth.User{Username: "bob", Email: "bob@example.com", DisplayName: "Bob"}
	u3 := &auth.User{Username: "charlie", Email: "charlie@example.com", DisplayName: "Charlie"}
	_ = authRepo.CreateUser(ctx, u1)
	_ = authRepo.CreateUser(ctx, u2)
	_ = authRepo.CreateUser(ctx, u3)

	repo := NewMemoryRepository(authRepo)
	svc := NewService(repo, authRepo)

	// 1. Alice creates organization "forge-crew"
	org, err := svc.CreateOrganization(ctx, u1.ID, CreateOrgRequest{
		Name:        "Forge Crew",
		Slug:        "forge-crew",
		Description: "Primary org for tests",
	})
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}
	if org.Role != OrgRoleOwner {
		t.Errorf("creator should have owner role, got %s", org.Role)
	}

	// 2. Duplicate slug should be rejected
	_, err = svc.CreateOrganization(ctx, u2.ID, CreateOrgRequest{
		Name: "Forge Crew Duplicate",
		Slug: "forge-crew",
	})
	if err == nil {
		t.Fatalf("expected error on duplicate org slug")
	}

	// 3. Alice adds Bob as Member
	bobMember, err := svc.AddMember(ctx, u1.ID, false, "forge-crew", AddOrgMemberRequest{
		Username: "bob",
		Role:     OrgRoleMember,
	})
	if err != nil {
		t.Fatalf("failed to add Bob to org: %v", err)
	}
	if bobMember.Role != OrgRoleMember {
		t.Errorf("expected Bob to be member, got %s", bobMember.Role)
	}

	// 4. Bob (regular member) cannot create a team
	_, err = svc.CreateTeam(ctx, u2.ID, false, "forge-crew", CreateTeamRequest{
		Name: "Frontend Team",
		Slug: "frontend",
	})
	if err == nil {
		t.Errorf("expected error when regular member tries to create team")
	}

	// 5. Alice (owner) creates team "frontend"
	team, err := svc.CreateTeam(ctx, u1.ID, false, "forge-crew", CreateTeamRequest{
		Name:        "Frontend Team",
		Slug:        "frontend",
		Description: "UI/UX engineers",
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	if team.CurrentUserRole != TeamRoleMaintainer {
		t.Errorf("team creator should be maintainer, got %s", team.CurrentUserRole)
	}

	// 6. Charlie cannot be added to team because he is NOT yet an org member
	_, err = svc.AddTeamMember(ctx, u1.ID, false, "forge-crew", "frontend", AddTeamMemberRequest{
		Username: "charlie",
		Role:     TeamRoleMember,
	})
	if err == nil || err != ErrUserNotOrgMember {
		t.Errorf("expected ErrUserNotOrgMember when adding non-org user to team, got %v", err)
	}

	// 7. Add Bob to team frontend
	tm, err := svc.AddTeamMember(ctx, u1.ID, false, "forge-crew", "frontend", AddTeamMemberRequest{
		Username: "bob",
		Role:     TeamRoleMember,
	})
	if err != nil {
		t.Fatalf("failed to add Bob to team: %v", err)
	}
	if tm.Username != "bob" {
		t.Errorf("expected bob, got %s", tm.Username)
	}

	// 8. Cannot demote or remove the last owner (Alice)
	err = svc.UpdateMemberRole(ctx, u1.ID, false, "forge-crew", "alice", OrgRoleAdmin)
	if err == nil || err != ErrCannotRemoveLastOwner {
		t.Errorf("expected ErrCannotRemoveLastOwner when demoting last owner, got %v", err)
	}

	err = svc.RemoveMember(ctx, u1.ID, false, "forge-crew", "alice")
	if err == nil || err != ErrCannotRemoveLastOwner {
		t.Errorf("expected ErrCannotRemoveLastOwner when removing last owner, got %v", err)
	}
}
