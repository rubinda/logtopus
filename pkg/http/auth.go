package http

import (
	"crypto"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Error message for token validation.
var (
	ErrUnexpectedSigningMethod error = fmt.Errorf("unexpected signing method")
	ErrTokenInvalid            error = fmt.Errorf("invalid token")
	ErrTokenMalformed          error = fmt.Errorf("can't parse token")
	ErrTokenEmpty              error = fmt.Errorf("token is empty")
	ErrTokenMissing            error = fmt.Errorf(`missing "Token" in header`)
)

// eventSourceClaims represents JWT payload.
type eventSourceClaims struct {
	IssuedTo string `json:"issuedTo"`
	jwt.RegisteredClaims
}

// JWTAuthority is a token issuer and validator.
type JWTAuthority struct {
	privateKey crypto.PrivateKey
	publicKey  crypto.PublicKey
}

// NewJWTAuthority returns a new JWT token issuer and validator.
func NewJWTAuthority(privateKeyFilePath, publicKeyFilePath string) (*JWTAuthority, error) {
	keyBytes, err := os.ReadFile(privateKeyFilePath)
	if err != nil {
		return nil, err
	}
	pubKeyBytes, err := os.ReadFile(publicKeyFilePath)
	if err != nil {
		return nil, err
	}
	key, err := jwt.ParseEdPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, err
	}
	pubKey, err := jwt.ParseEdPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return &JWTAuthority{key, pubKey}, nil
}

// ValidateToken checks if given token is valid with our issuer. Returns error when token is invalid.
func (jwtAuth *JWTAuthority) ValidateToken(tokenStr string) (*jwt.Token, error) {
	if tokenStr == "" {
		return nil, ErrTokenEmpty
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, ErrUnexpectedSigningMethod
		}
		return jwtAuth.publicKey, nil
	})
	if err != nil {
		return nil, ErrTokenMalformed
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	return token, nil
}

// IssueToken generates a new token for a given event source.
func (jwtAuth *JWTAuthority) IssueToken(requestee string) (string, error) {
	claims := eventSourceClaims{
		requestee,
		jwt.RegisteredClaims{
			// TODO:
			//  - tokens shouldn't last this long for security concerns (replay attacks, hijacking etc.)
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(jwtAuth.privateKey)
}
