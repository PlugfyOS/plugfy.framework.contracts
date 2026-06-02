package agent_test

import (
	"testing"

	"github.com/PlugfyOS/plugfy-common/agent"
)

// TestPrimitiveMetaInterface asserts every concrete primitive embeds
// PrimitiveMeta and therefore satisfies the Primitive interface — the contract
// the registry/resolver rely on to treat declarations uniformly.
func TestPrimitiveMetaInterface(t *testing.T) {
	prims := []agent.Primitive{
		agent.AgentDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindAgentDef, ID: "a"}},
		agent.ToolDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindToolDef, ID: "t"}},
		agent.Skill{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindSkill, ID: "s"}},
		agent.PromptDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindPromptDef, ID: "p"}},
		agent.InstructionDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindInstructionDef, ID: "i"}},
		agent.ModelRef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindModelRef, ID: "m"}},
		agent.ConfigDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindConfigDef, ID: "c"}},
		agent.GuardrailDef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindGuardrailDef, ID: "g"}},
		agent.SubAgentRef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindSubAgentRef, ID: "sa"}},
		agent.MCPServerRef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindMCPServerRef, ID: "mcp"}},
		agent.A2APeerRef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindA2APeerRef, ID: "a2a"}},
		agent.ConnectorRef{PrimitiveMeta: agent.PrimitiveMeta{Kind: agent.KindConnectorRef, ID: "conn"}},
	}
	if len(prims) != len(agent.AllPrimitiveKinds()) {
		t.Fatalf("constructed %d primitives but catalog enumerates %d kinds", len(prims), len(agent.AllPrimitiveKinds()))
	}
	for _, p := range prims {
		if p.GetMeta().ID == "" {
			t.Fatalf("primitive %T lost its envelope id", p)
		}
	}
}

// TestAllPrimitiveKinds asserts the catalog enumerates the twelve canonical
// kinds with the AgentDef leading.
func TestAllPrimitiveKinds(t *testing.T) {
	kinds := agent.AllPrimitiveKinds()
	if len(kinds) != 12 {
		t.Fatalf("AllPrimitiveKinds = %d, want 12", len(kinds))
	}
	if kinds[0] != agent.KindAgentDef {
		t.Fatalf("AllPrimitiveKinds[0] = %q, want %q", kinds[0], agent.KindAgentDef)
	}
}
