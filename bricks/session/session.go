package session

import (
	"time"
)

type AuthenticationMethods struct {
	Aal         string    `json:"aal"`
	CompletedAt time.Time `json:"completed_at"`
	Method      string    `json:"method"`
}

type Devices struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	Location  string `json:"location"`
	UserAgent string `json:"user_agent"`
}

type RecoveryAddresses struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	Value     string    `json:"value"`
	Via       string    `json:"via"`
}

type Name struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

type Traits struct {
	Email string `json:"email"`
	Name  Name   `json:"name"`
}

type VerifiableAddresses struct {
	CreatedAt  time.Time `json:"created_at"`
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
	Value      string    `json:"value"`
	Verified   bool      `json:"verified"`
	VerifiedAt time.Time `json:"verified_at"`
	Via        string    `json:"via"`
}

type Identity struct {
	CreatedAt           time.Time             `json:"created_at"`
	ID                  string                `json:"id"`
	MetadataPublic      interface{}           `json:"metadata_public"`
	RecoveryAddresses   []RecoveryAddresses   `json:"recovery_addresses"`
	SchemaID            string                `json:"schema_id"`
	SchemaURL           string                `json:"schema_url"`
	State               string                `json:"state"`
	StateChangedAt      time.Time             `json:"state_changed_at"`
	Traits              Traits                `json:"traits"`
	UpdatedAt           time.Time             `json:"updated_at"`
	VerifiableAddresses []VerifiableAddresses `json:"verifiable_addresses"`
}

type Session struct {
	Active                      bool                    `json:"active"`
	AuthenticatedAt             time.Time               `json:"authenticated_at"`
	AuthenticationMethods       []AuthenticationMethods `json:"authentication_methods"`
	AuthenticatorAssuranceLevel string                  `json:"authenticator_assurance_level"`
	Devices                     []Devices               `json:"devices"`
	ExpiresAt                   time.Time               `json:"expires_at"`
	ID                          string                  `json:"id"`
	Identity                    Identity                `json:"identity"`
	IssuedAt                    time.Time               `json:"issued_at"`
}
