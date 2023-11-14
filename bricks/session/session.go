package session

import (
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
)

type AuthenticationMethods struct {
	Aal         string    `json:"aal" mapstructure:"aal"`
	CompletedAt time.Time `json:"completed_at" mapstructure:"completed_at"`
	Method      string    `json:"method" mapstructure:"method"`
}

type Devices struct {
	ID        string `json:"id" mapstructure:"id"`
	IPAddress string `json:"ip_address" mapstructure:"ip_address"`
	Location  string `json:"location" mapstructure:"location"`
	UserAgent string `json:"user_agent" mapstructure:"user_agent"`
}

type RecoveryAddresses struct {
	CreatedAt time.Time `json:"created_at" mapstructure:"created_at"`
	ID        string    `json:"id" mapstructure:"id"`
	UpdatedAt time.Time `json:"updated_at" mapstructure:"updated_at"`
	Value     string    `json:"value" mapstructure:"value"`
	Via       string    `json:"via" mapstructure:"via"`
}

type Name struct {
	First string `json:"first" mapstructure:"first"`
	Last  string `json:"last" mapstructure:"last"`
}

type Traits struct {
	Email string `json:"email" mapstructure:"email"`
	Name  Name   `json:"name" mapstructure:"name"`
}

type VerifiableAddresses struct {
	CreatedAt  time.Time `json:"created_at" mapstructure:"created_at"`
	ID         string    `json:"id" mapstructure:"id"`
	Status     string    `json:"status" mapstructure:"status"`
	UpdatedAt  time.Time `json:"updated_at" mapstructure:"updated_at"`
	Value      string    `json:"value" mapstructure:"value"`
	Verified   bool      `json:"verified" mapstructure:"verified"`
	VerifiedAt time.Time `json:"verified_at" mapstructure:"verified_at"`
	Via        string    `json:"via" mapstructure:"via"`
}

type Identity struct {
	CreatedAt           time.Time             `json:"created_at" mapstructure:"created_at"`
	ID                  string                `json:"id" mapstructure:"id"`
	MetadataPublic      interface{}           `json:"metadata_public" mapstructure:"metadata_public"`
	RecoveryAddresses   []RecoveryAddresses   `json:"recovery_addresses" mapstructure:"recovery_addresses"`
	SchemaID            string                `json:"schema_id" mapstructure:"schema_id"`
	SchemaURL           string                `json:"schema_url" mapstructure:"schema_url"`
	State               string                `json:"state" mapstructure:"state"`
	StateChangedAt      time.Time             `json:"state_changed_at" mapstructure:"state_changed_at"`
	Traits              Traits                `json:"traits" mapstructure:"traits"`
	UpdatedAt           time.Time             `json:"updated_at" mapstructure:"updated_at"`
	VerifiableAddresses []VerifiableAddresses `json:"verifiable_addresses" mapstructure:"verifiable_addresses"`
}

type Session struct {
	Active                      bool                    `json:"active" mapstructure:"active"`
	AuthenticatedAt             time.Time               `json:"authenticated_at" mapstructure:"authenticated_at"`
	AuthenticationMethods       []AuthenticationMethods `json:"authentication_methods" mapstructure:"authentication_methods"`
	AuthenticatorAssuranceLevel string                  `json:"authenticator_assurance_level" mapstructure:"authenticator_assurance_level"`
	Devices                     []Devices               `json:"devices" mapstructure:"devices"`
	ExpiresAt                   time.Time               `json:"expires_at" mapstructure:"expires_at"`
	ID                          string                  `json:"id" mapstructure:"id"`
	Identity                    Identity                `json:"identity" mapstructure:"identity"`
	IssuedAt                    time.Time               `json:"issued_at" mapstructure:"issued_at"`
}

func FromTokenClaims(claims jwt.MapClaims) Session {
	result := Session{}
	sess, ok := claims["session"]
	if !ok {
		return result
	}
	data, ok := sess.(map[string]interface{})
	if !ok {
		slog.Error("failed convert session claims to map[string]interface{}")
		return result
	}

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToTimeHookFunc(time.RFC3339),
		WeaklyTypedInput: true,
		Result:           &result,
	})
	if err != nil {
		slog.Error("failed create mapstructure decoder", slog.Any("err", err))
		return result
	}
	if err := d.Decode(data); err != nil {
		slog.Error("failed decode session claims", slog.Any("err", err))
		return result
	}
	return result
}
