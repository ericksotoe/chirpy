package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	// params := &argon2id.Params{
	// 	Memory:      128 * 1024,
	// 	Iterations:  4,
	// 	Parallelism: uint8(runtime.NumCPU()),
	// 	SaltLength:  16,
	// 	KeyLength:   32,
	// }

	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedJWT, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedJWT, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// Prepare a claims struct to be populated by ParseWithClaims
	claims := jwt.RegisteredClaims{}

	// Parse and validate the token, filling in the claims struct
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		// Return the same key type that was used to sign the token
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	// Ensure the token was issued by our application
	if claims.Issuer != "chirpy" {
		return uuid.Nil, errors.New("invalid issuer")
	}

	// Extract the subject (user ID string) from the token claims
	idString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	// Parse the subject string into a UUID
	id, err := uuid.Parse(idString)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenString := headers.Get("Authorization")
	if tokenString == "" {
		return "", errors.New("The header has no Authorization parameter")
	}

	wordToStrip := "Bearer "
	hasPrefix := strings.HasPrefix(tokenString, wordToStrip)
	if !hasPrefix {
		return "", errors.New("The Authorization header has no Bearer")
	}

	strippedTokenString := strings.TrimPrefix(tokenString, wordToStrip)
	strippedTokenString = strings.TrimSpace(strippedTokenString)

	return strippedTokenString, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	encodedString := hex.EncodeToString(key)
	return encodedString, nil
}
