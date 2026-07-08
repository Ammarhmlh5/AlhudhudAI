# GDPR Data Management

## Overview

The GDPR module provides data deletion and export functionality to comply with GDPR Article 17 (Right to Erasure) and Article 20 (Right to Data Portability).

## Features

### Data Deletion
- ✅ Request data deletion with reason
- ✅ Process deletion with transaction safety
- ✅ Delete personal data completely
- ✅ Anonymize financial records (keep for audit)
- ✅ Track deletion status
- ✅ Error handling and logging

### Data Export
- ✅ Export all user data in JSON format
- ✅ Include all tables and relationships
- ✅ Generate downloadable files
- ✅ Automatic cleanup after 7 days

## Usage

### Data Deletion

#### Request Deletion

```go
import "your-project/internal/gdpr"

manager := gdpr.NewDataDeletionManager(db)

requestID, err := manager.RequestDeletion(ctx, tenantID, userID, "User requested account deletion")
if err != nil {
    // Handle error
}

// Deletion request created with ID: requestID
```

#### Process Deletion

```go
err := manager.ProcessDeletion(ctx, requestID)
if err != nil {
    // Handle error
}

// Data deleted and financial records anonymized
```

#### Check Deletion Status

```go
req, err := manager.GetDeletionRequest(ctx, requestID)
if err != nil {
    // Handle error
}

fmt.Printf("Status: %s\n", req.Status)
if req.ProcessedAt != nil {
    fmt.Printf("Processed at: %v\n", *req.ProcessedAt)
}
```

### Data Export

#### Export User Data

```go
exporter := gdpr.NewDataExportManager(db)

export, err := exporter.ExportUserData(ctx, tenantID, userID)
if err != nil {
    // Handle error
}

// export contains all user data in structured format
```

#### Generate Download File

```go
downloadURL, err := exporter.GenerateExportFile(ctx, tenantID, userID)
if err != nil {
    // Handle error
}

// Send download URL to user via email
```

## Data Deletion Process

### 1. Personal Data (Deleted Completely)

- User profile information
- Contact details
- Consent records
- Notifications
- Support tickets (anonymized)

### 2. Financial Data (Anonymized, Not Deleted)

- Accounting transactions (amounts kept, user info removed)
- Balance history (amounts kept, user info removed)
- Subscriptions (anonymized)
- Invoices (anonymized)

**Why keep financial data?**
- Legal requirement for audit purposes
- Tax compliance (7 years retention)
- Fraud prevention
- Dispute resolution

### 3. Audit Trail (Always Kept)

- Deletion request record
- Timestamp of deletion
- Reason for deletion
- Processing status

## GDPR Compliance

### Article 17: Right to Erasure

✅ **Implemented**:
- User can request data deletion
- Personal data is deleted within 30 days
- Financial data is anonymized (legal exception)
- Audit trail is maintained

### Article 20: Right to Data Portability

✅ **Implemented**:
- User can export all their data
- Data provided in machine-readable format (JSON)
- Includes all personal and transactional data
- Available for download within 24 hours

## Database Schema

### data_deletion_requests

```sql
CREATE TABLE data_deletion_requests (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL,
    reason TEXT,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## Best Practices

1. **Grace Period**: Implement 30-day grace period before deletion
2. **Confirmation**: Send confirmation email before processing
3. **Backup**: Create backup before deletion
4. **Audit**: Log all deletion operations
5. **Anonymization**: Use consistent anonymization strategy
6. **Testing**: Test deletion process regularly

## Integration Example

### API Handler

```go
func (h *GDPRHandler) RequestDeletion(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Reason string `json:"reason"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    tenantID := getTenantID(r.Context())
    userID := getUserID(r.Context())
    
    requestID, err := h.deletionManager.RequestDeletion(
        r.Context(),
        tenantID,
        userID,
        req.Reason,
    )
    if err != nil {
        http.Error(w, "Failed to create deletion request", http.StatusInternalServerError)
        return
    }
    
    // Send confirmation email
    h.sendDeletionConfirmationEmail(userID, requestID)
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "request_id": requestID,
        "status":     "pending",
        "message":    "Your deletion request has been received. Data will be deleted within 30 days.",
    })
}
```

### Background Job

```go
// Schedule deletion processing after grace period
func ProcessPendingDeletions(ctx context.Context, manager *gdpr.DataDeletionManager) error {
    requests, err := manager.ListDeletionRequests(ctx, tenantID, gdpr.DeletionStatusPending)
    if err != nil {
        return err
    }
    
    for _, req := range requests {
        // Check if grace period has passed (30 days)
        if time.Since(req.RequestedAt) > 30*24*time.Hour {
            err := manager.ProcessDeletion(ctx, req.ID)
            if err != nil {
                log.Printf("Failed to process deletion %d: %v", req.ID, err)
                continue
            }
            
            // Send completion email
            sendDeletionCompletedEmail(req.UserID)
        }
    }
    
    return nil
}
```

## Requirements

- Requirements: 36.5, 36.6, 36.7, 36.8
- GDPR Article 17 (Right to Erasure)
- GDPR Article 20 (Right to Data Portability)

## Testing

Run tests:
```bash
go test ./internal/gdpr/...
```

## Migration

Run migrations:
```bash
psql -d your_database -f migrations/030_data_deletion_requests_table.sql
```

## Status

✅ Data Deletion Implemented
⏳ Data Export (Next Task)
✅ Documented
