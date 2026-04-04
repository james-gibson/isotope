package isotope

import "time"

// Registration is the request body for POST /isotope/register.
type Registration struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol,omitempty"`
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
}
