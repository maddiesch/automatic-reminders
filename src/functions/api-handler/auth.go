package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
)

func apiTokenForAccount(a *auto.Account) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: time.Now().AddDate(0, 0, 90).Unix(),
		Issuer:    "autorem://api/v1",
		Audience:  "autorem://api/v1",
		Subject:   a.ID,
		IssuedAt:  time.Now().Unix(),
	})

	return token.SignedString([]byte(auto.Secrets().Signing))
}

const (
	contextUserIDKey = "_auid"
	token
)

// Authenticate performs authentication
func Authenticate(c *gin.Context) {
	accountID, err := performAuthentication(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"Error": err.Error()})
		return
	}

	c.Set(contextUserIDKey, accountID)

	c.Header("X-User-ID", accountID)
}

func performAuthentication(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid authorization header")
	}

	switch strings.ToLower(strings.TrimSpace(parts[0])) {
	case "bearer":
		return getAccountIDAndValidateToken(strings.TrimSpace(parts[1]))
	default:
		return "", fmt.Errorf("Invalid authorization type")
	}
}

func getAccountIDAndValidateToken(t string) (string, error) {
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(auto.Secrets().Signing), nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("Invalid token")
	}

	{ // Workaround for a but that doesn't let standard claims to be decoded automatically.
		data, _ := json.Marshal(token.Claims)
		claims := jwt.StandardClaims{}
		err := json.Unmarshal(data, &claims)
		if err != nil {
			return "", err
		}

		if err := claims.Valid(); err != nil {
			return "", err
		}

		token.Claims = claims
	}

	claims, ok := token.Claims.(jwt.StandardClaims)
	if !ok {
		return "", errors.New("Invalid token (claims)")
	}

	return claims.Subject, nil
}
