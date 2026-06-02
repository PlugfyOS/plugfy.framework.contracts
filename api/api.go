// Package api defines the api.v1 route-contribution contract: the pure-data
// description of the HTTP routes a unit contributes to the PlugfyOS API host.
//
// This is a declaration, not an implementation. A route-provider (spi.KindAPI)
// returns a [RouteSet] of [RouteContribution]s; the platform API host (L6)
// reads that data and mounts the concrete handlers. Crucially, this package
// imports no HTTP machinery — no net/http, no router — so the L1 baseplate
// stays a stdlib-only leaf and route declarations can be marshaled, diffed,
// catalogued and signed as plain data independently of any server runtime.
//
// The wire shape (JSON tags below) is the stable contract between the unit
// that declares routes and the host that mounts them; it is frozen by the
// golden ABI test and a JSON round-trip test in this package.
package api

// AuthScope is the authorization requirement a route declares. The API host
// enforces it before dispatching to the handler. It is string-backed so the
// value travels verbatim on the wire and in catalogues.
type AuthScope string

const (
	// AuthNone marks a public route: no authentication is required
	// (health checks, OpenAPI documents, public discovery endpoints).
	AuthNone AuthScope = "none"
	// AuthUser marks a route that requires an authenticated principal
	// scoped to the active organization/project (the default for most
	// tenant-facing endpoints).
	AuthUser AuthScope = "user"
	// AuthAdmin marks a route that requires platform-administrator
	// authority (installation, capability binding, tenant lifecycle).
	AuthAdmin AuthScope = "admin"
)

// Route is a single HTTP route declaration. It carries everything the API host
// needs to mount the endpoint and everything a catalogue/OpenAPI generator
// needs to describe it, but no handler: the handler is resolved by the host
// from OperationID at mount time.
type Route struct {
	// Method is the uppercase HTTP method ("GET", "POST", "PUT",
	// "PATCH", "DELETE").
	Method string `json:"method"`
	// Path is the route path relative to the contribution BasePath
	// (e.g. "/{id}"). It MAY contain host-router path parameters.
	Path string `json:"path"`
	// AuthScope is the authorization requirement enforced before
	// dispatch.
	AuthScope AuthScope `json:"authScope"`
	// OperationID is the stable, unique identifier the host uses to bind
	// this declaration to a concrete handler and that OpenAPI uses as the
	// operationId. Convention: lowerCamelCase, verb-first
	// (e.g. "listInstalledModules").
	OperationID string `json:"operationId"`
	// Summary is a short human-readable description for documentation.
	Summary string `json:"summary,omitempty"`
	// RequestSchemaRef references the JSON-schema component describing the
	// request body (e.g. "#/components/schemas/InstallRequest"). Empty for
	// bodyless requests.
	RequestSchemaRef string `json:"requestSchemaRef,omitempty"`
	// ResponseSchemaRef references the JSON-schema component describing the
	// success response body. Empty when the route returns no body.
	ResponseSchemaRef string `json:"responseSchemaRef,omitempty"`
	// Streaming marks a route whose response is a server-sent-event
	// (text/event-stream) stream; the gateway dispatches it via
	// supervisorv1.InvokeStream and relays frames, instead of the unary
	// Invoke.
	Streaming bool `json:"streaming,omitempty"`
}

// RouteContribution is one logical group of routes a unit contributes, sharing
// a Group label and a BasePath. The API host mounts each Route under
// BasePath+Route.Path.
type RouteContribution struct {
	// Group is the logical name of this route group (e.g. "installed",
	// "capabilities"). Used for documentation tags and host-side grouping.
	Group string `json:"group"`
	// BasePath is the path prefix shared by every Route in the group
	// (e.g. "/api/v1/installed"). It MUST begin with "/".
	BasePath string `json:"basePath"`
	// Routes are the individual route declarations under BasePath.
	Routes []Route `json:"routes"`
}

// RouteSet is the resolved bundle a route-provider returns: the full set of
// route contributions a unit offers to the API host in one negotiation. It is
// the top-level wire document the host reads when mounting a unit's API.
type RouteSet struct {
	// Provider is the name of the unit contributing these routes (matches
	// the provider's spi.Provider Name).
	Provider string `json:"provider"`
	// Contributions are the route groups this provider offers.
	Contributions []RouteContribution `json:"contributions"`
}
