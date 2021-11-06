package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"
	"github.com/terrycain/actions-cache-server/pkg/s"
)

func parseToken(token *jwt.Token) (interface{}, error) {
	// Check we have been signed by an acceptable algorithm
	wellKnownData, err := GetWellKnownData()
	if err != nil {
		return nil, err
	}
	found := false
	for _, alg := range wellKnownData.SignatureTypes {
		if alg == token.Header["alg"] {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("signature type %s is not valid", token.Header["alg"])
	}

	keyIDInterface, exists := token.Header["x5t"]
	if !exists {
		return nil, errors.New("x5t claim in header doesnt exist")
	}
	keyID, ok := keyIDInterface.(string)
	if !ok {
		return nil, errors.New("x5t claim in header is not a string")
	}

	keyData, err := LookupKey(keyID)
	return keyData, err
}

func (h *Handlers) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing Authorization header"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		token := parts[1]

		// parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parser := jwt.Parser{
			SkipClaimsValidation: h.Debug, // So this is deprecated but i can't seem to use NewParser, get undefined method error
		}
		parsedToken, err := parser.Parse(token, parseToken)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to validate token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to validate token"})
			c.Abort()
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok || !parsedToken.Valid {
			log.Warn().Msg("Failed to validate token, something wrong with claims")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to validate token"})
			c.Abort()
			return
		}

		scopeString := claims["ac"].(string)
		scopeList := make([]s.Scope, 0)
		if err = json.Unmarshal([]byte(scopeString), &scopeList); err != nil {
			log.Warn().Err(err).Msg("Failed to validate token, issue with scope claims")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to validate token"})
			c.Abort()
			return
		}
		if len(scopeList) < 1 {
			log.Warn().Err(err).Msg("Failed to validate token, no scopes")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			c.Abort()
			return
		}

		c.Set("scopes", scopeList)
		c.Next()
	}
}
