// This file declares the CANONICAL Agent Hub contract: the data plane that
// catalogs — for the General Agent AND any other agent/app — every agent
// primitive as a scoped, ACL'd, declarative definition, plus the AgentHub
// capability port that registers, lists and RESOLVES those primitives into a
// runnable, permission-/limit-filtered agent plan the engine consumes.
//
// It is a sibling of ai.go (the Assistant/Event contract) and follows the same
// discipline: declaration-only value types and interfaces, importing nothing but
// plugfy-common/spi, so it crosses unit boundaries cleanly without leaking ADK or
// any driver type. The General Agent is just an AgentDef (Official:true) resolved
// through this exact pipeline — no privileged code path.
//
// Maps 1:1 onto the AGENT-LAYER.md catalog (§2.2.1): the twelve primitives — skills,
// sub-agents, tools, prompts, models, configs, system instructions, guardrails,
// MCP servers/toolsets, A2A peers, connectors — plus the assembled AgentDef. The
// Hub stores DECLARATIONS; the AI unit's domain/engine remains the execution plane.
// Resolution applies the Agent-Gateway seam (§2.5): ACL + limits + guardrails fold
// in here so candidates the caller may not use never reach the model.
package agent

import (
	"context"
	"errors"

	commonspi "github.com/PlugfyOS/plugfy.framework.contracts/spi"
)

// ErrPrimitiveNotFound is returned by [AgentHub.Get]/[AgentHub.Resolve] when a
// referenced primitive does not exist in (or inherited into) the active scope.
var ErrPrimitiveNotFound = errors.New("agent-hub: primitive not found")

// KindAgentHub is the SPI kind under which an [AgentHub] self-registers with the
// platform registry. Like KindAI it is a host capability (a system service), not
// an edition-swapped driver, so its value is chosen to keep registry lookups
// collision-free with plugfy-common/spi's canonical kinds.
const KindAgentHub commonspi.Kind = "agent-hub"

// PrimitiveKind enumerates the Hub catalog entries. Each value names one typed,
// versioned, scoped declaration the Hub stores and the resolver folds into the
// ADK constructs the AI unit's domain/engine consumes.
type PrimitiveKind string

// The twelve catalog primitives (AGENT-LAYER.md §2.2.1). KindAgentDef is the
// assembled agent itself; the rest are the parts it references.
const (
	KindAgentDef       PrimitiveKind = "agent_def"       // an assembled, composable agent
	KindSkill          PrimitiveKind = "skill"           // a curated tool+prompt+guardrail bundle
	KindSubAgentRef    PrimitiveKind = "sub_agent_ref"   // a local AgentDef or remote A2A peer used as a sub-agent
	KindToolDef        PrimitiveKind = "tool_def"        // a callable tool (function / MCP / connector-as-tool)
	KindPromptDef      PrimitiveKind = "prompt_def"      // a versioned, templated instruction/system prompt
	KindModelRef       PrimitiveKind = "model_ref"       // a model capability + options
	KindConfigDef      PrimitiveKind = "config_def"      // non-secret inheritable agent config (KV)
	KindInstructionDef PrimitiveKind = "instruction_def" // policy/system instructions distinct from the prompt
	KindGuardrailDef   PrimitiveKind = "guardrail_def"   // input/output policy (injection screen, PII, output cap, allowlist)
	KindMCPServerRef   PrimitiveKind = "mcp_server_ref"  // a remote MCP server/toolset
	KindA2APeerRef     PrimitiveKind = "a2a_peer_ref"    // a remote A2A agent (Agent-Card URL)
	KindConnectorRef   PrimitiveKind = "connector_ref"   // a connector source exposed as tool and/or RAG data-source
)

// AllPrimitiveKinds is the catalog's full set, in declaration order. It lets
// stores/registries iterate every kind without hard-coding the list.
func AllPrimitiveKinds() []PrimitiveKind {
	return []PrimitiveKind{
		KindAgentDef, KindSkill, KindSubAgentRef, KindToolDef, KindPromptDef,
		KindModelRef, KindConfigDef, KindInstructionDef, KindGuardrailDef,
		KindMCPServerRef, KindA2APeerRef, KindConnectorRef,
	}
}

// ScopeType is the kind of org-tree node a primitive is scoped to. The Hub owns
// this value type — it mirrors the marketplace/identity scope vocabulary by
// contract, NOT by import — so it stays decoupled while modelling scope-keyed,
// inheritable declarations.
type ScopeType string

const (
	// ScopePlatform is the root scope: a primitive visible everywhere (official
	// system primitives, e.g. the General Agent, live here).
	ScopePlatform ScopeType = "platform"
	// ScopeOrg scopes a primitive to an organization; it is inherited downward by
	// every descendant org and project.
	ScopeOrg ScopeType = "org"
	// ScopeProject scopes a primitive to a single project (a virtual desktop).
	ScopeProject ScopeType = "project"
)

// Scope points at one org-tree node a primitive belongs to.
//
// OrgID is the parent organization of a ScopeProject scope, when known. It does
// NOT change the node a primitive is stored AT (that is Type+ID); it only lets
// the inheritance chain walk project -> org -> platform so an org-level
// primitive is visible from a child project (AGENT-LAYER.md §2.2 — "inherited
// down the org tree"). It is empty for org/platform scopes and ignored by the
// storage key (scopeKey/scopeRow).
type Scope struct {
	Type  ScopeType `json:"type"`
	ID    string    `json:"id,omitempty"`    // empty for ScopePlatform
	OrgID string    `json:"orgId,omitempty"` // parent org of a project scope, for inheritance only
}

// ReadWriteClass classifies a tool/connector operation's side effects. Following
// the Antigravity safest-default policy (AGENT-LAYER.md §1.4/§2.5), write-class
// primitives are excluded at resolution unless the agent is explicitly granted
// write in the active scope.
type ReadWriteClass string

const (
	// ClassRead is a side-effect-free operation (search, fetch, list). Read-only
	// is the default capability of every agent.
	ClassRead ReadWriteClass = "read"
	// ClassWrite mutates external state (create, update, delete). Excluded unless
	// the agent holds a write grant in scope.
	ClassWrite ReadWriteClass = "write"
)

// PrimitiveMeta is the envelope common to every Hub declaration: its kind,
// identity, version, scope, ACL and the entitlement keys it consumes. The
// resolver reads ACL + Limits to gate the primitive for a calling identity.
type PrimitiveMeta struct {
	Kind        PrimitiveKind  `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version,omitempty"`
	Scope       Scope          `json:"scope"`
	ACL         []string       `json:"acl,omitempty"`    // identity subjects/groups (system-identity); empty = public
	Limits      []string       `json:"limits,omitempty"` // entitlement keys this primitive consumes (system-entitlements)
	Config      map[string]any `json:"config,omitempty"` // free-form non-secret options
}

// GetMeta returns the envelope. Implemented by every primitive so the registry
// and resolver treat them uniformly through the [Primitive] interface.
func (m PrimitiveMeta) GetMeta() PrimitiveMeta { return m }

// Primitive is satisfied by every catalog declaration: it exposes its common
// envelope. Concrete primitives embed [PrimitiveMeta]. The registry stores and
// the resolver folds Primitives without knowing their concrete type beyond the
// kind discriminator.
type Primitive interface {
	GetMeta() PrimitiveMeta
}

// ----------------------------------------------------------------------------
// The twelve catalog primitives.
// ----------------------------------------------------------------------------

// PromptDef is a versioned, templated instruction/system prompt. Template is
// rendered with Variables (and the request context) into the agent Instruction.
type PromptDef struct {
	PrimitiveMeta
	Template  string            `json:"template"`
	Variables map[string]string `json:"variables,omitempty"`
}

// InstructionDef is a system/operating-rule text the model must obey, distinct
// from the user-visible prompt. It is prepended to the rendered Instruction.
type InstructionDef struct {
	PrimitiveMeta
	Policy string `json:"policy"`
}

// ModelRef names a model capability plus generation options. ModelName is the
// registry name of the model backend; the resolver passes the options through to
// the engine, which binds them to the consumer-owned ModelPort.
type ModelRef struct {
	PrimitiveMeta
	ModelName   string  `json:"modelName"`
	Temperature float32 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
}

// ToolKind classifies how a ToolDef is realized into an ADK tool.
type ToolKind string

const (
	// ToolFunction is an in-process function tool (the host registers the
	// implementation in the engine's tool registry by Name).
	ToolFunction ToolKind = "function"
	// ToolMCP is a tool exposed by a referenced MCP server (MCPRef names it).
	ToolMCP ToolKind = "mcp"
	// ToolConnector is a connector operation exposed as a tool (ConnectorRef
	// names the source; Operation names the op, e.g. "search").
	ToolConnector ToolKind = "connector"
)

// ToolDef declares a callable tool: a function tool, an MCP tool or a
// connector-as-tool. Schema is the JSON Schema (draft 2020-12) of its arguments;
// Class gates read/write exposure; RequiredScopes/Limits gate ACL/quota.
type ToolDef struct {
	PrimitiveMeta
	ToolKind      ToolKind       `json:"toolKind"`
	Schema        []byte         `json:"schema,omitempty"`
	Class         ReadWriteClass `json:"class"`
	MCPRef        string         `json:"mcpRef,omitempty"`        // for ToolMCP: the MCPServerRef id
	ConnectorRef  string         `json:"connectorRef,omitempty"`  // for ToolConnector: the ConnectorRef id
	Operation     string         `json:"operation,omitempty"`     // for ToolConnector: the operation name
	RequiredScope string         `json:"requiredScope,omitempty"` // an extra identity scope the caller must hold
}

// Skill is a named capability bundle — a curated set of tools + a prompt +
// guardrails — advertised as an A2A "skill"/Agent-Card entry and the unit of
// reuse a builder drags in. The resolver expands it into its referenced parts.
type Skill struct {
	PrimitiveMeta
	ToolIDs      []string `json:"toolIds,omitempty"`
	PromptID     string   `json:"promptId,omitempty"`
	GuardrailIDs []string `json:"guardrailIds,omitempty"`
}

// SubAgentRefKind classifies a sub-agent reference as a local AgentDef or a
// remote A2A peer.
type SubAgentRefKind string

const (
	// SubAgentLocal references another AgentDef in the Hub (wired via agenttool).
	SubAgentLocal SubAgentRefKind = "local"
	// SubAgentA2A references a remote A2A peer (wired via remoteagent.NewA2A).
	SubAgentA2A SubAgentRefKind = "a2a"
)

// SubAgentRef references another agent usable as a sub-agent: a local AgentDef
// (RefID is its id) or a remote A2A peer (RefID is an A2APeerRef id).
type SubAgentRef struct {
	PrimitiveMeta
	RefKind SubAgentRefKind `json:"refKind"`
	RefID   string          `json:"refId"`
}

// GuardrailDef is an input/output policy: blocked patterns, PII redaction, an
// output cap and a tool-use allowlist (our "Model Armor" + semantic governance
// analog, AGENT-LAYER.md §2.5). The resolver compiles it into before/after
// model & tool callbacks; the policy itself is declared by system-security and
// only REFERENCED here.
type GuardrailDef struct {
	PrimitiveMeta
	BlockedPatterns []string `json:"blockedPatterns,omitempty"` // substrings/keywords blocked on model input
	RedactPatterns  []string `json:"redactPatterns,omitempty"`  // substrings redacted (PII) from model output
	MaxOutputChars  int      `json:"maxOutputChars,omitempty"`  // 0 = uncapped
	ToolAllowlist   []string `json:"toolAllowlist,omitempty"`   // when non-empty, only these tool names may run
}

// MCPTransport classifies an MCP server's transport.
type MCPTransport string

const (
	// MCPStdio launches a local command and speaks MCP over stdio.
	MCPStdio MCPTransport = "stdio"
	// MCPSSE connects to a remote MCP server over Server-Sent Events.
	MCPSSE MCPTransport = "sse"
	// MCPStreamableHTTP connects over the streamable-HTTP MCP transport.
	MCPStreamableHTTP MCPTransport = "streamable_http"
)

// MCPServerRef declares a remote MCP server/toolset: transport, endpoint, an
// auth secret reference (resolved at call time via system-security, NEVER stored
// here) and an allowed-tool allowlist.
type MCPServerRef struct {
	PrimitiveMeta
	Transport    MCPTransport `json:"transport"`
	URL          string       `json:"url,omitempty"`     // for sse/streamable_http
	Command      string       `json:"command,omitempty"` // for stdio
	Args         []string     `json:"args,omitempty"`    // for stdio
	SecretRef    string       `json:"secretRef,omitempty"`
	AllowedTools []string     `json:"allowedTools,omitempty"`
}

// A2APeerRef declares a remote A2A agent by its Agent-Card URL, an auth secret
// reference and its advertised skills. The resolver adds it as a sub-agent when
// permitted (remoteagent.NewA2A).
type A2APeerRef struct {
	PrimitiveMeta
	AgentCardURL string   `json:"agentCardUrl"`
	SecretRef    string   `json:"secretRef,omitempty"`
	Skills       []string `json:"skills,omitempty"`
}

// ConnectorRef declares a platform-provider-connector source exposed as a tool
// and/or a RAG data-source. Driver is the connector driver name (e.g. "fs",
// "jira"); SecretRef resolves its credentials at call time via system-security.
type ConnectorRef struct {
	PrimitiveMeta
	Driver     string   `json:"driver"`
	SecretRef  string   `json:"secretRef,omitempty"`
	Operations []string `json:"operations,omitempty"` // operations exposed as connector-as-tools
	AsDataset  bool     `json:"asDataset,omitempty"`  // also expose as a RAG data-source
}

// ConfigDef is non-secret agent config (KV), inheritable by scope. Values fold
// into the resolved agent's run-time options.
type ConfigDef struct {
	PrimitiveMeta
	Values map[string]any `json:"values,omitempty"`
}

// AgentDef is the published, composable agent definition — the data form of an
// ADK agent. The resolver folds it (and its referenced parts) into a
// ResolvedAgent, which the AI unit's domain/engine translates into an
// llmagent.Config. The General Agent is an AgentDef with Official:true.
type AgentDef struct {
	PrimitiveMeta
	ModelRef      string   `json:"modelRef"`                // a ModelRef id
	InstructionID string   `json:"instructionId,omitempty"` // a PromptDef id (the user-visible prompt)
	SystemRuleIDs []string `json:"systemRuleIds,omitempty"` // InstructionDef ids (prepended policy)
	ToolIDs       []string `json:"toolIds,omitempty"`       // ToolDef ids
	SkillIDs      []string `json:"skillIds,omitempty"`      // Skill ids (expanded into tools/prompt/guardrails)
	SubAgentIDs   []string `json:"subAgentIds,omitempty"`   // SubAgentRef ids (local AgentDefs or A2A peers)
	MCPServerIDs  []string `json:"mcpServerIds,omitempty"`  // MCPServerRef ids
	ConnectorIDs  []string `json:"connectorIds,omitempty"`  // ConnectorRef ids
	GuardrailIDs  []string `json:"guardrailIds,omitempty"`  // GuardrailDef ids
	ConfigIDs     []string `json:"configIds,omitempty"`     // ConfigDef ids
	WriteGranted  bool     `json:"writeGranted,omitempty"`  // when true, write-class tools may be exposed
	Official      bool     `json:"official,omitempty"`      // true for the General Agent (system app)
	Deterministic bool     `json:"deterministic,omitempty"` // prefer deterministic (pipeline/workflow) routing
	PipelineID    string   `json:"pipelineId,omitempty"`    // the named pipeline for deterministic routing
}

// ----------------------------------------------------------------------------
// Resolution: the heart of the Hub.
// ----------------------------------------------------------------------------

// ResolveRequest carries the {agent, project, identity} context the resolver
// folds an AgentDef under: which agent to assemble, the active project (scope),
// and the calling identity's subjects/scopes so ACL + entitlement filtering run
// for THIS caller in THIS project. Limits/guardrails tighten as the foundation
// units land; permissive defaults keep it runnable offline.
type ResolveRequest struct {
	AgentID    string   `json:"agentId"`
	ProjectID  string   `json:"projectId"`
	OrgID      string   `json:"orgId,omitempty"`
	UserID     string   `json:"userId"`
	UserScopes []string `json:"userScopes,omitempty"` // subject + group identifiers (ACL)
}

// ResolvedTool is a ToolDef that survived ACL/limit/read-write filtering, ready
// for the engine to materialize into an ADK tool.
type ResolvedTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      []byte         `json:"schema,omitempty"`
	Kind        ToolKind       `json:"kind"`
	Class       ReadWriteClass `json:"class"`
	MCPRef      string         `json:"mcpRef,omitempty"`
	ConnectorID string         `json:"connectorId,omitempty"`
	Operation   string         `json:"operation,omitempty"`
}

// ResolvedSubAgent is a sub-agent reference that survived filtering.
type ResolvedSubAgent struct {
	RefKind      SubAgentRefKind `json:"refKind"`
	RefID        string          `json:"refId"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	AgentCardURL string          `json:"agentCardUrl,omitempty"` // for SubAgentA2A
}

// ResolvedMCP is an MCPServerRef that survived ACL/limit filtering, ready for
// the engine (in the `mcp` build) to attach as a remote toolset. It carries the
// transport/endpoint and the SecretRef the credential plane resolves at call
// time — the secret VALUE is never carried here (resolved via CredentialPort at
// the composition root, AGENT-LAYER.md §2.3).
type ResolvedMCP struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Transport    MCPTransport `json:"transport"`
	URL          string       `json:"url,omitempty"`
	Command      string       `json:"command,omitempty"`
	Args         []string     `json:"args,omitempty"`
	SecretRef    string       `json:"secretRef,omitempty"`
	AllowedTools []string     `json:"allowedTools,omitempty"`
	Scope        Scope        `json:"scope"`
}

// ResolvedConnector is a ConnectorRef that survived ACL/limit filtering. The
// engine/host builds a connector-as-tool for each declared Operation through the
// ConnectorPort (the credential is resolved from SecretRef at call time). It is
// the source declaration behind the agent's ToolConnector ResolvedTools.
type ResolvedConnector struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Driver     string   `json:"driver"`
	SecretRef  string   `json:"secretRef,omitempty"`
	Operations []string `json:"operations,omitempty"`
	AsDataset  bool     `json:"asDataset,omitempty"`
	Scope      Scope    `json:"scope"`
}

// ResolvedGuardrail is a compiled guardrail policy the engine turns into
// before/after model & tool callbacks.
type ResolvedGuardrail struct {
	BlockedPatterns []string `json:"blockedPatterns,omitempty"`
	RedactPatterns  []string `json:"redactPatterns,omitempty"`
	MaxOutputChars  int      `json:"maxOutputChars,omitempty"`
	ToolAllowlist   []string `json:"toolAllowlist,omitempty"`
}

// ResolvedAgent is the permission/limit-filtered, ready-to-run agent
// description the engine consumes. It carries NO ADK types — the AI unit's
// domain/engine translates it into an llmagent.Config (instruction + allowed
// tools + allowed sub-agents + guardrail callbacks + model name/options).
type ResolvedAgent struct {
	AgentID     string              `json:"agentId"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Instruction string              `json:"instruction"`
	ModelName   string              `json:"modelName,omitempty"`
	Temperature float32             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"maxTokens,omitempty"`
	Tools       []ResolvedTool      `json:"tools,omitempty"`
	SubAgents   []ResolvedSubAgent  `json:"subAgents,omitempty"`
	MCPServers  []ResolvedMCP       `json:"mcpServers,omitempty"`
	Connectors  []ResolvedConnector `json:"connectors,omitempty"`
	Guardrails  []ResolvedGuardrail `json:"guardrails,omitempty"`
	Config      map[string]any      `json:"config,omitempty"`
	// Dropped records primitives excluded by ACL/limit/read-write filtering, for
	// observability and the builder UX ("blocked-by-default-unless-permitted").
	Dropped []DroppedPrimitive `json:"dropped,omitempty"`
}

// DropReason classifies why a primitive was excluded at resolution.
type DropReason string

const (
	// DropACL means the caller's identity is not on the primitive's ACL.
	DropACL DropReason = "acl"
	// DropLimit means an entitlement/quota for the primitive is exhausted/denied.
	DropLimit DropReason = "limit"
	// DropWriteClass means a write-class tool was excluded (no write grant).
	DropWriteClass DropReason = "write_class"
	// DropMissing means a referenced primitive id does not exist in scope.
	DropMissing DropReason = "missing"
	// DropGuardrailError means a referenced GuardrailDef failed to COMPILE through
	// the GuardrailPort (a security-side failure, distinct from a dangling ref):
	// the guardrail cannot be enforced so it is dropped, and the run continues
	// under whatever other guardrails compiled.
	DropGuardrailError DropReason = "guardrail_error"
)

// DroppedPrimitive records one excluded primitive and why.
type DroppedPrimitive struct {
	Kind   PrimitiveKind `json:"kind"`
	ID     string        `json:"id"`
	Reason DropReason    `json:"reason"`
}

// AgentHub is the OWNED capability port: register, look up, list and RESOLVE
// every Hub primitive. It embeds plugfy-common/spi's Provider so it registers and
// health-checks like any other capability (under KindAgentHub). Consumers
// (app-agent-builder, the orchestrator) resolve it at runtime by capability,
// never by direct import.
type AgentHub interface {
	commonspi.Provider

	// Register upserts a primitive declaration (keyed by kind+id+scope).
	Register(ctx context.Context, p Primitive) error
	// Get returns a primitive by kind + id, searched up the scope chain implied
	// by the active scope (project -> org -> platform). Returns ErrPrimitiveNotFound
	// when absent.
	Get(ctx context.Context, kind PrimitiveKind, id string, sc Scope) (Primitive, error)
	// List returns every primitive of a kind visible in the given scope (direct +
	// inherited from ancestor scopes).
	List(ctx context.Context, kind PrimitiveKind, sc Scope) ([]Primitive, error)
	// Remove deletes a primitive declaration. Removing an absent primitive is a
	// no-op (idempotent).
	Remove(ctx context.Context, kind PrimitiveKind, id string, sc Scope) error
	// Resolve materializes a permission/limit-filtered ResolvedAgent for the
	// {agent, project, identity} request: it reads the AgentDef, expands its
	// skills, renders its instruction, and filters its tools/sub-agents/connectors
	// through ACL + entitlements + read/write class for the calling identity.
	Resolve(ctx context.Context, req ResolveRequest) (ResolvedAgent, error)
}
