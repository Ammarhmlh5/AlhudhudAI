# Consent Management System

## Overview

The Consent Management system provides GDPR/CCPA compliant consent tracking for user agreements and data processing permissions.

## Features

- ✅ Record user consent with full audit trail
- ✅ Track consent versions
- ✅ Support multiple consent types
- ✅ Revoke consent functionality
- ✅ IP address and user agent tracking
- ✅ Tenant isolation with RLS

## Consent Types

1. **terms_of_service**: User agreement to Terms of Service
2. **privacy_policy**: User agreement to Privacy Policy
3. **marketing**: User consent for marketing communications
4. **data_processing**: User consent for data processing

## Usage

### Initialize Manager

```go
import "your-project/internal/consent"

manager := consent.NewManager(db)
```

### Record Consent

```go
req := &consent.ConsentRequest{
    UserID:    userID,
    Type:      consent.ConsentTermsOfService,
    Version:   "1.0",
    Granted:   true,
    IPAddress: "192.168.1.1",
    UserAgent: "Mozilla/5.0...",
}

err := manager.RecordConsent(ctx, tenantID, req)
```

### Check Consent

```go
granted, err := manager.CheckConsent(ctx, tenantID, userID, consent.ConsentPrivacyPolicy)
if err != nil {
    // Handle error
}

if !granted {
    // User has not consented
}
```

### Revoke Consent

```go
err := manager.RevokeConsent(ctx, tenantID, userID, consent.ConsentMarketing)
```

### Get User Consents

```go
consents, err := manager.GetUserConsents(ctx, tenantID, userID)
for _, c := range consents {
    fmt.Printf("Type: %s, Granted: %v, At: %v\n", c.Type, c.Granted, c.GrantedAt)
}
```

## Database Schema

```sql
CREATE TABLE user_consents (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    consent_type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## Compliance

### GDPR Requirements

- ✅ **Article 7**: Conditions for consent
- ✅ **Article 13**: Information to be provided
- ✅ **Article 17**: Right to erasure
- ✅ **Article 20**: Right to data portability

### CCPA Requirements

- ✅ **Right to Know**: Users can view their consent history
- ✅ **Right to Delete**: Consent can be revoked
- ✅ **Right to Opt-Out**: Marketing consent can be withdrawn

## Best Practices

1. **Always record consent before processing data**
2. **Include version numbers** for all consent documents
3. **Track IP address and user agent** for audit purposes
4. **Allow users to revoke consent** at any time
5. **Keep consent records** for at least 7 years

## Integration with Registration

```go
// During user registration
func RegisterUser(ctx context.Context, req *RegistrationRequest) error {
    // Create user account
    userID, err := createUser(ctx, req)
    if err != nil {
        return err
    }

    // Record consent
    consentReq := &consent.ConsentRequest{
        UserID:    userID,
        Type:      consent.ConsentTermsOfService,
        Version:   "1.0",
        Granted:   req.AcceptedTerms,
        IPAddress: req.IPAddress,
        UserAgent: req.UserAgent,
    }

    err = consentManager.RecordConsent(ctx, tenantID, consentReq)
    if err != nil {
        return err
    }

    return nil
}
```

## Requirements

- Requirements: 36.3, 36.4
- GDPR Compliance
- CCPA Compliance

## Testing

Run tests:
```bash
go test ./internal/consent/...
```

## Migration

Run migration:
```bash
psql -d your_database -f migrations/029_user_consents_table.sql
```

## Status

✅ Implemented
✅ Tested
✅ Documented
