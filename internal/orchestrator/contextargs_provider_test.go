package orchestrator

import (
	"context"
	"testing"

	"github.com/opentalon/opentalon/internal/actor"
	"github.com/opentalon/opentalon/internal/profile"
	"github.com/opentalon/opentalon/pkg/plugin/contextargs"
)

// TestDefaultContextArgProviders_GroupEntity locks in the security-relevant
// wiring: group_id / entity_id resolve from the authenticated actor context,
// and are EMPTY when the actor has no group/identity — so a consuming plugin
// fails closed rather than inheriting another tenant's scope.
func TestDefaultContextArgProviders_GroupEntity(t *testing.T) {
	// o is unused by the group/entity closures (only the allowed_* providers
	// deref it), so a nil orchestrator is safe here.
	providers := defaultContextArgProviders(nil, nil)

	group := providers[contextargs.GroupID]
	entity := providers[contextargs.EntityID]
	if group == nil || entity == nil {
		t.Fatal("group_id/entity_id providers not registered")
	}

	// Empty context: no actor, no group. Must resolve to "" (fail-closed).
	if got := group(context.Background(), contextargs.GroupID); got != "" {
		t.Errorf("group_id on empty ctx = %q, want empty", got)
	}
	if got := entity(context.Background(), contextargs.EntityID); got != "" {
		t.Errorf("entity_id on empty ctx = %q, want empty", got)
	}

	// Populated context: values flow from the actor scope.
	ctx := actor.WithGroupID(actor.WithActor(context.Background(), "e1"), "g1")
	if got := group(ctx, contextargs.GroupID); got != "g1" {
		t.Errorf("group_id = %q, want g1", got)
	}
	if got := entity(ctx, contextargs.EntityID); got != "e1" {
		t.Errorf("entity_id = %q, want e1", got)
	}

	// Actor set but no group (profile-less dev): entity flows, group empty.
	ctxNoGroup := actor.WithActor(context.Background(), "console:u1")
	if got := group(ctxNoGroup, contextargs.GroupID); got != "" {
		t.Errorf("group_id without group = %q, want empty", got)
	}
	if got := entity(ctxNoGroup, contextargs.EntityID); got != "console:u1" {
		t.Errorf("entity_id = %q, want console:u1", got)
	}
}

// TestDefaultContextArgProviders_RunLabels locks in the run-label wiring:
// interaction_kind / system_source resolve off the profile as stored and are
// EMPTY when no profile is on the context — so an unlabelled run reaches a
// plugin as an absent arg rather than a guessed one. (The verifier never
// produces an empty Kind; the last case pins that the provider does not
// paper over a hand-built profile that has none.)
func TestDefaultContextArgProviders_RunLabels(t *testing.T) {
	providers := defaultContextArgProviders(nil, nil)

	kind := providers[contextargs.InteractionKind]
	source := providers[contextargs.SystemSource]
	if kind == nil || source == nil {
		t.Fatal("interaction_kind/system_source providers not registered")
	}

	cases := []struct {
		name       string
		profile    *profile.Profile
		wantKind   string
		wantSource string
	}{
		{name: "no profile", profile: nil},
		{
			name:     "chat turn carries a kind and no source",
			profile:  &profile.Profile{Kind: profile.KindChat},
			wantKind: profile.KindChat,
		},
		{
			name:       "system run carries both labels",
			profile:    &profile.Profile{Kind: profile.KindSystem, SystemSource: "nightly_report"},
			wantKind:   profile.KindSystem,
			wantSource: "nightly_report",
		},
		{
			name:    "a profile built without a kind is reported as is",
			profile: &profile.Profile{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.profile != nil {
				ctx = profile.WithProfile(ctx, tc.profile)
			}
			if got := kind(ctx, contextargs.InteractionKind); got != tc.wantKind {
				t.Errorf("interaction_kind = %q, want %q", got, tc.wantKind)
			}
			if got := source(ctx, contextargs.SystemSource); got != tc.wantSource {
				t.Errorf("system_source = %q, want %q", got, tc.wantSource)
			}
		})
	}
}
