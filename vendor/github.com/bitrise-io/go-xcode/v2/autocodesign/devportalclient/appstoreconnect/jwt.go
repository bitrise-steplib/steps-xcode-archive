package appstoreconnect

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// signToken signs the JWT token with the given .p8 private key content
func signToken(token *jwt.Token, privateKeyContent []byte) (string, error) {
	block, _ := pem.Decode(privateKeyContent)
	if block == nil {
		return "", errors.New("failed to parse private key as a PEM format")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token, private key format is invalid: %v", err)
	}

	privateKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return "", errors.New("not a private key")
	}

	return token.SignedString(privateKey)
}

// createToken creates a jwt.Token for the Apple API
func createToken(keyID string, issuerID string) *jwt.Token {
	payload := claims{
		IssuerID:   issuerID,
		Expiration: time.Now().Add(jwtDuration).Unix(),
		Audience:   "appstoreconnect-v1",
	}

	// registers headers: alg = ES256 and typ = JWT
	token := jwt.NewWithClaims(jwt.SigningMethodES256, payload)

	header := token.Header
	header["kid"] = keyID

	return token
}

// claims represents the JWT payload for the Apple API
type claims struct {
	IssuerID   string `json:"iss"`
	Expiration int64  `json:"exp"`
	Audience   string `json:"aud"`
}

// Valid implements the jwt.Claims interface
func (c claims) Valid() error {
	return nil
}
