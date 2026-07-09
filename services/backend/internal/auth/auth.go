package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
)

// HashKey hashes a plain key for secure storage/comparison
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ValidateAccountingKey checks if the provided key belongs to an active tenant
func ValidateAccountingKey(apiKey string) (uuid.UUID, error) {
	hashed := HashKey(apiKey)

	var tenantID uuid.UUID
	var isActive bool
	query := `SELECT tenant_id, is_active FROM accounting_keys WHERE key_hash = $1`

	err := db.DB.QueryRow(query, hashed).Scan(&tenantID, &isActive)
	if err != nil {
		return uuid.Nil, errors.New("invalid accounting key")
	}

	if !isActive {
		return uuid.Nil, errors.New("accounting key is inactive")
	}

	return tenantID, nil
}

// ValidateDevicePair checks if the device belongs to the tenant and is authorized
func ValidateDevicePair(tenantID uuid.UUID, pairKey string) (uuid.UUID, error) {
	hashed := HashKey(pairKey)

	var deviceID uuid.UUID
	var status string
	query := `SELECT id, status FROM paired_devices WHERE tenant_id = $1 AND device_pair_key = $2`

	err := db.DB.QueryRow(query, tenantID, hashed).Scan(&deviceID, &status)
	if err != nil {
		return uuid.Nil, errors.New("invalid or unauthorized device key")
	}

	return deviceID, nil
}
