package isotope

import "time"

// Registration is the request body for POST /isotope/register.
type Registration struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol,omitempty"`
	// Attestation is the cluster receipt that raises this registration above the
	// floor. Without it the endpoint is self-declared and the rung is 2,
	// whatever the registry says (isotope#23).
	Attestation *Attestation `json:"attestation,omitempty"`
}

// Record describes a registered isotope instance with its assigned trust rung.
type Record struct {
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Endpoint     string    `json:"endpoint"`
	Protocol     string    `json:"protocol"`
	TrustRung    int       `json:"trust_rung"`
	RungName     string    `json:"rung_name"`
	RegisteredAt time.Time `json:"registered_at"`
	// Evidence is why the rung is what it is, in the verifier's words. A rung an
	// operator cannot explain is a rung they cannot audit.
	Evidence []string `json:"evidence,omitempty"`
	// AttestedBy names the cluster that vouched for this endpoint, empty when
	// nothing did.
	AttestedBy string `json:"attested_by,omitempty"`
}
