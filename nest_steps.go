package isotope

// ---- the Nest vocabulary ----
//
// A Nest feature is read two ways, like the workspace-VFS vocabulary:
//
//   * CanonicalizeGherkinToIsotope answers "is this the same test?" — it hashes
//     the surface (the scenario steps) and DROPS the Background. The port is
//     deliberately absent everywhere: a port is a deployment-collision decision
//     an app announces, never a feature's declaration.
//   * Define answers "what does this test ask for?" — it reads both the surface
//     (what the feature serves) and the Background (what it presupposes).
//
// The two readings agree because the presuppositions live in Background, which
// the canon drops: a presupposition is a ledger edge, not part of the
// dependent's content identity. "It becomes the path that matters" — the
// route/resolver/stream IS the feature's identity; the guard/model/feature it
// names is a dependency, recorded alongside but not hashed in.

// Nest transport constants — the feature's declared requirement, keyed by the
// endpoint kind that requires it. These are the same names the transport
// registry (feature-transport.json) uses, so the two cannot drift.
const (
	TransportHTTP     = "http"
	TransportGraphQL  = "graphql"
	TransportSSE      = "sse"
	TransportJSONRPC  = "json-rpc"
	TransportUDP      = "udp"
	TransportSocketIO = "socket.io"
	TransportOTLP     = "otlp"
	TransportTFTP     = "tftp"
)

// varValue returns the named placeholder's captured value.
func varValue(key string) func(map[string]string) string {
	return func(v map[string]string) string { return v[key] }
}

// constValue ignores the placeholders and returns a fixed value (a transport).
func constValue(s string) func(map[string]string) string {
	return func(map[string]string) string { return s }
}

// NestStepDefinitions returns the definitions for the Nest decomposition
// vocabulary. The generators emit steps in this vocabulary; an app reads the
// resulting Spec with ByKind to announce what it supports (surface + transport)
// and what it assumes (presuppositions).
//
// Each surface step declares two things: the endpoint's kind + value, and the
// transport it requires. An app reads ByKind("transport") to know whether it
// can host the feature, and ByKind("route")/ByKind("resolver")/… to know what
// it must serve. The port is nowhere in either.
//
// Presupposition steps live in Background (dropped by the canon, so they do
// not change the isotope): "the feature {name}", "the Prisma model {name}",
// "the {feature} guard", "the {feature} service". An app reads those kinds to
// know what it must also mount, transitively.
func NestStepDefinitions() []StepDefinition {
	return []StepDefinition{
		// ---- surface: what the feature serves ----

		// HTTP route -> path + http
		{Pattern: `the route "{path}"`, Kind: "route", Value: varValue("path")},
		{Pattern: `the route "{path}"`, Kind: "transport", Value: constValue(TransportHTTP)},

		// GraphQL resolver -> operation + graphql
		{Pattern: `the GraphQL resolver "{op}"`, Kind: "resolver", Value: varValue("op")},
		{Pattern: `the GraphQL resolver "{op}"`, Kind: "transport", Value: constValue(TransportGraphQL)},

		// SSE stream -> path + sse
		{Pattern: `the SSE stream "{path}"`, Kind: "sse_stream", Value: varValue("path")},
		{Pattern: `the SSE stream "{path}"`, Kind: "transport", Value: constValue(TransportSSE)},

		// JSON-RPC method -> method + json-rpc
		{Pattern: `the JSON-RPC method "{method}"`, Kind: "rpc_method", Value: varValue("method")},
		{Pattern: `the JSON-RPC method "{method}"`, Kind: "transport", Value: constValue(TransportJSONRPC)},

		// UDP listener -> event + udp
		{Pattern: `the UDP listener "{event}"`, Kind: "udp_listener", Value: varValue("event")},
		{Pattern: `the UDP listener "{event}"`, Kind: "transport", Value: constValue(TransportUDP)},

		// Socket.IO gateway event
		{Pattern: `the Socket.IO event "{event}"`, Kind: "gateway_event", Value: varValue("event")},
		{Pattern: `the Socket.IO event "{event}"`, Kind: "transport", Value: constValue(TransportSocketIO)},

		// OTLP export -> signal + otlp
		{Pattern: `the OTLP export "{signal}"`, Kind: "telemetry", Value: varValue("signal")},
		{Pattern: `the OTLP export "{signal}"`, Kind: "transport", Value: constValue(TransportOTLP)},

		// TFTP server -> path + tftp
		{Pattern: `the TFTP server "{path}"`, Kind: "tftp_server", Value: varValue("path")},
		{Pattern: `the TFTP server "{path}"`, Kind: "transport", Value: constValue(TransportTFTP)},

		// scheduled job (no wire transport)
		{Pattern: `the scheduled job "{job}"`, Kind: "cron", Value: varValue("job")},

		// ---- presuppositions: what the feature assumes (Background) ----

		// cross-feature dependency -> the presupposed feature, by name
		{Pattern: `the feature "{name}"`, Kind: "feature", Value: varValue("name")},

		// Prisma schema dependency -> the model the resolver presupposes
		{Pattern: `the Prisma model "{name}"`, Kind: "model", Value: varValue("name")},

		// auth guard / auth service — the two auth dependencies, distinguished
		// by kind so a generator knows which to wire (the guard or the service)
		{Pattern: `the {feature} guard`, Kind: "guard", Value: varValue("feature")},
		{Pattern: `the {feature} service`, Kind: "service", Value: varValue("feature")},
	}
}
