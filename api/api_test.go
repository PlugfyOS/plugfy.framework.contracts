package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestRouteSetJSONRoundTrip locks the api.v1 wire shape: a fully-populated
// RouteSet must survive a marshal/unmarshal round trip unchanged, and the
// emitted JSON must use the exact field names downstream consumers depend on.
func TestRouteSetJSONRoundTrip(t *testing.T) {
	in := RouteSet{
		Provider: "platform-api",
		Contributions: []RouteContribution{
			{
				Group:    "installed",
				BasePath: "/api/v1/installed",
				Routes: []Route{
					{
						Method:            "GET",
						Path:              "",
						AuthScope:         AuthUser,
						OperationID:       "listInstalledModules",
						Summary:           "List installed modules",
						ResponseSchemaRef: "#/components/schemas/InstalledIndex",
					},
					{
						Method:           "POST",
						Path:             "",
						AuthScope:        AuthAdmin,
						OperationID:      "installModule",
						Summary:          "Install a module",
						RequestSchemaRef: "#/components/schemas/InstallRequest",
					},
					{
						Method:      "GET",
						Path:        "/health",
						AuthScope:   AuthNone,
						OperationID: "installedHealth",
					},
				},
			},
		},
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out RouteSet
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip mismatch:\n in = %+v\nout = %+v", in, out)
	}
}

// TestRouteWireFieldNames pins the exact JSON keys so an accidental tag rename
// (a silent ABI break for every consumer of the contract) fails here.
func TestRouteWireFieldNames(t *testing.T) {
	r := Route{
		Method:            "POST",
		Path:              "/{id}",
		AuthScope:         AuthAdmin,
		OperationID:       "installModule",
		Summary:           "Install a module",
		RequestSchemaRef:  "#/req",
		ResponseSchemaRef: "#/resp",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	for _, key := range []string{
		"method", "path", "authScope", "operationId",
		"summary", "requestSchemaRef", "responseSchemaRef",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing wire field %q in %s", key, b)
		}
	}
	if got := m["authScope"]; got != string(AuthAdmin) {
		t.Errorf("authScope = %v, want %q", got, AuthAdmin)
	}
}

// TestAuthScopeTokens pins the string tokens of the AuthScope enum: these
// travel on the wire and in catalogues, so a value change is a breaking change.
func TestAuthScopeTokens(t *testing.T) {
	cases := map[AuthScope]string{
		AuthNone:  "none",
		AuthUser:  "user",
		AuthAdmin: "admin",
	}
	for scope, want := range cases {
		if string(scope) != want {
			t.Errorf("AuthScope token = %q, want %q", string(scope), want)
		}
	}
}

// TestOmitEmptyOptionalFields verifies the optional fields drop out of the wire
// form when empty, keeping declarations compact.
func TestOmitEmptyOptionalFields(t *testing.T) {
	b, err := json.Marshal(Route{
		Method:      "GET",
		Path:        "/health",
		AuthScope:   AuthNone,
		OperationID: "health",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"summary", "requestSchemaRef", "responseSchemaRef"} {
		if _, present := m[key]; present {
			t.Errorf("optional field %q should be omitted when empty: %s", key, b)
		}
	}
}
