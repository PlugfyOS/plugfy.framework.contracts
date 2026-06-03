// Package agent is the CANONICAL, public home of the Plugfy AI/Agent-Hub
// contracts: the Assistant/chat surface (Run request + streamed events) and the
// Agent Hub catalog (the twelve declarative agent primitives plus the resolver
// data plane). These are pure declaration-only value types and interfaces —
// importing nothing but plugfy-common/spi — so they cross unit boundaries
// cleanly without leaking ADK, a model driver or any other unit's
// implementation.
//
// # Why these contracts live in plugfy-common
//
// They are CONSUMER-FACING: an agent/app author (plugfy-adk-starter-pack,
// plugfy-examples and every Tier-3 unit) declares an [AgentDef], its [ToolDef]s
// and [Scope] without depending on the AI bounded-context implementation
// (system-ai). Hosting them on the public, stdlib-only leaf module (plugfy-common)
// lets authors reach the surface through the SDK (plugfy-sdk/agent re-exports
// this package) while the AI unit (system-ai) re-sources the very same types as
// thin back-compat aliases — so the canonical definition lives once, on the
// public surface, and every existing importer keeps compiling unchanged.
//
// The base Provider/Kind types are reused from plugfy-common/spi so every
// capability satisfies the same lifecycle contract; the Assistant capability
// registers under [KindAI] and the Agent Hub under [KindAgentHub].
//
// This file declares the Assistant/chat surface; hub.go declares the Agent Hub
// catalog and resolver. Both follow the same discipline: no imports beyond
// plugfy-common/spi.
package agent

import (
	"context"

	commonspi "github.com/PlugfyOS/plugfy.framework.contracts/spi"
)

// KindAI is the SPI kind under which an [Assistant] self-registers with the
// platform registry. It is not one of plugfy-common/spi's canonical provider
// kinds because the AI assistant is a host capability (a system service), not an
// edition-swapped driver; the value is chosen so registry lookups stay
// collision-free.
const KindAI commonspi.Kind = "ai"

// Role classifies the author of a chat message.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// RunRequest is a single chat turn submitted to the assistant. The caller's own
// identity and scopes travel with it so every downstream capability (RAG,
// tools) runs with the user's permissions and never broader.
type RunRequest struct {
	// ConversationID, when set, persists the turn into an existing conversation
	// and threads prior history into the prompt. Empty starts a fresh,
	// unpersisted turn.
	ConversationID string
	ProjectID      string
	// OrgID is the parent organization of ProjectID, when known. It is purely
	// ADDITIVE and OPTIONAL: callers that do not track the org tree (e.g.
	// platform-api today) leave it empty and behaviour is unchanged. When set, the
	// Agent Hub threads it into scope resolution so a primitive declared at the
	// org level is inherited down into the org's projects (the documented
	// "project -> org -> platform" inheritance, AGENT-LAYER.md §2.2). Consumers of
	// this contract may ignore it.
	OrgID  string
	UserID string
	// UserScopes are the subject + group identifiers used for ACL-aware RAG.
	UserScopes []string
	// Text is the user's prompt.
	Text string
}

// EventKind classifies a streamed assistant event.
type EventKind string

// Canonical assistant event kinds. The stream always terminates with a single
// Done event (or an Error followed by Done).
const (
	EventToken    EventKind = "token"     // a fragment of the answer
	EventTool     EventKind = "tool"      // a tool/app invocation occurred
	EventCitation EventKind = "citation"  // a RAG grounding source
	EventUISchema EventKind = "ui_schema" // a server-driven UI component to render
	EventDone     EventKind = "done"      // end of run
	EventError    EventKind = "error"     // a run-level error
)

// Event is a single streamed unit of an assistant run.
type Event struct {
	Kind EventKind `json:"kind"`
	Text string    `json:"text,omitempty"`
	Data any       `json:"data,omitempty"`
}

// Assistant is the AI capability port consumers resolve at runtime: it runs a
// chat turn and streams events (tokens, tool calls, citations, server-driven
// UI). Implementations compose a model backend, the knowledge/RAG plane and an
// optional orchestration engine behind the consumer-owned ports declared in the
// AI unit's domain package — the contract here stays free of those collaborators.
type Assistant interface {
	commonspi.Provider
	// Run executes a single chat turn and streams events. The returned channel
	// is closed when the run completes.
	Run(ctx context.Context, req RunRequest) (<-chan Event, error)
}
