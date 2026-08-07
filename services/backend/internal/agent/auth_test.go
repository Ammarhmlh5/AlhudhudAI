package agent

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAccessTokenAndValidate(t *testing.T) {
	agentID := uuid.New()

	token, err := GenerateAccessToken(agentID, agentID, "agent", "admin")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := validateToken(token)
	require.NoError(t, err)
	assert.Equal(t, agentID.String(), claims.UserID)
	assert.Equal(t, agentID.String(), claims.AgentID)
	assert.Equal(t, "agent", claims.UserType)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "access", claims.Type)
}

func TestValidateRefreshToken(t *testing.T) {
	userID := uuid.New()

	token, err := GenerateRefreshToken(userID, "employee")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, "refresh", claims.Type)
}

func TestValidateTokenRejectsAccessTokenAsRefresh(t *testing.T) {
	agentID := uuid.New()

	access, err := GenerateAccessToken(agentID, agentID, "agent", "admin")
	require.NoError(t, err)

	_, err = ValidateRefreshToken(access)
	assert.Error(t, err)
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	agentID := uuid.New()

	// Manually build an already-expired access token
	now := time.Now()
	claims := TokenClaims{
		UserID:   agentID.String(),
		AgentID:  agentID.String(),
		Role:     "admin",
		UserType: "agent",
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(JWTKey))
	require.NoError(t, err)

	_, err = validateToken(signed)
	assert.Error(t, err)
}
