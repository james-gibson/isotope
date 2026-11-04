# Nest decomposition — example features

Gherkin capture of a TypeScript/NestJS prototype's services, decomposed one
feature file per service. These are example specs for the isotope canon: each
feature is transport-agnostic (a service declares its endpoints and the
transport each requires, never a port), and its dependencies are declared as
`Background:` presuppositions — readable for meaning, dropped from the isotope.

Directories are apps, files are services:

- `api-app/` — the main NestJS app (GraphQL + REST + SSE + JSON-RPC)
- `api-admin/` — the admin app and its UDP sensor services
- `api-auth/` — the auth app
- `api-runtime/` — the P2P runtime app
- `dht-tracker/` — the standalone Socket.IO tracker
- `api-cli/` — the GraphQL CLI client
- `ui/` — the website build and runtime

Provenance: captured from a private prototype. Site-specific identity
(hostname, domain, project name) has been scrubbed — "the prototype" in the
headers names the source without naming the house. The private flavour is a
variant of the generic spec: the private token process uses a non-standard
prefix where the generic spec says the standard "Bearer".

Read with `NestStepDefinitions` (see `nest_steps.go`).
