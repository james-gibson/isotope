package isotope

import (
	"reflect"
	"testing"
)

const nestSource = `Feature: The note service stores and links notes

  Background:
    Given the Prisma model "Note"
    Given the Prisma model "Link"
    Given the auth guard

  Scenario: the note surface
    Given the GraphQL resolver "addNote"
    Given the GraphQL resolver "getJSX"

  Scenario: the stream surface
    Given the SSE stream "/hookers/sse"
    Given the scheduled job "ping"
`

const nestUserSource = `Feature: The user service checks tokens and records users

  Background:
    Given the auth service
    Given the Prisma model "User"
    Given the feature "message-bus"

  Scenario: the user surface
    Given the GraphQL resolver "isValidAuth"
    Given the GraphQL resolver "addUser"
`

func TestDefineReadsNestSurface(t *testing.T) {
	spec, err := Define(nestSource, NestStepDefinitions()...)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if spec.Feature != "The note service stores and links notes" {
		t.Errorf("feature = %q", spec.Feature)
	}
	if got, want := spec.ByKind("resolver"), []string{"addNote", "getJSX"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resolvers = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("sse_stream"), []string{"/hookers/sse"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sse streams = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("cron"), []string{"ping"}; !reflect.DeepEqual(got, want) {
		t.Errorf("crons = %v, want %v", got, want)
	}
	// the transport a resolver/stream requires — what an app must support to load it
	if got, want := spec.ByKind("transport"), []string{"graphql", "sse"}; !reflect.DeepEqual(got, want) {
		t.Errorf("transports = %v, want %v", got, want)
	}
	// presuppositions live in Background: read for meaning, but not hashed into
	// the isotope (the canon drops Background)
	if got, want := spec.ByKind("model"), []string{"Link", "Note"}; !reflect.DeepEqual(got, want) {
		t.Errorf("models = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("guard"), []string{"auth"}; !reflect.DeepEqual(got, want) {
		t.Errorf("guards = %v, want %v", got, want)
	}
}

func TestBackgroundDoesNotChangeIsotope(t *testing.T) {
	withBG := `Feature: The note service stores and links notes

  Background:
    Given the Prisma model "Note"
    Given the auth guard

  Scenario: the note surface
    Given the GraphQL resolver "addNote"
`
	withoutBG := `Feature: The note service stores and links notes

  Scenario: the note surface
    Given the GraphQL resolver "addNote"
`
	key := []byte("seed-42")
	a, err := CanonicalizeGherkinToIsotope(withBG, key, "nest", 1)
	if err != nil {
		t.Fatalf("withBG: %v", err)
	}
	b, err := CanonicalizeGherkinToIsotope(withoutBG, key, "nest", 1)
	if err != nil {
		t.Fatalf("withoutBG: %v", err)
	}
	if a.Raw != b.Raw {
		t.Errorf("Background changed the isotope: %q != %q", a.Raw, b.Raw)
	}
}

func TestDefineReadsNestPresuppositions(t *testing.T) {
	spec, err := Define(nestUserSource, NestStepDefinitions()...)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if got, want := spec.ByKind("resolver"), []string{"addUser", "isValidAuth"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resolvers = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("model"), []string{"User"}; !reflect.DeepEqual(got, want) {
		t.Errorf("models = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("service"), []string{"auth"}; !reflect.DeepEqual(got, want) {
		t.Errorf("services = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("feature"), []string{"message-bus"}; !reflect.DeepEqual(got, want) {
		t.Errorf("features = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("transport"), []string{"graphql"}; !reflect.DeepEqual(got, want) {
		t.Errorf("transports = %v, want %v", got, want)
	}
}
