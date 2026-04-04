package isotope

import "time"

// IsotopeRegistration is the request body for POST /isotope/register.
type IsotopeRegistration struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol,omitempty"`
}

// IsotopeRecord describes a registered isotope instance with its assigned trust rung.
type IsotopeRecord struct {
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Endpoint     string    `json:"endpoint"`
	Protocol     string    `json:"protocol"`
	TrustRung    int       `json:"trust_rung"`
	RungName     string    `json:"rung_name"`
	RegisteredAt time.Time `json:"registered_at"`
}
