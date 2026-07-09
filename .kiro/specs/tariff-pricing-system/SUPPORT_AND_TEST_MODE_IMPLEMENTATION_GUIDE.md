# دليل تنفيذ نظام الدعم ووضع الاختبار

## نظرة عامة

هذا الدليل يغطي تنفيذ المهام 39 (نظام الدعم الفني) و 40 (وضع الاختبار). هذه المهام اختيارية ويمكن تنفيذها تدريجياً.

---

## المهمة 39: نظام الدعم الفني

### 39.1 إنشاء جدول support_tickets ⏳

**Migration**: `services/backend/migrations/030_support_tickets_table.sql`

```sql
CREATE TABLE IF NOT EXISTS support_tickets (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    ticket_number VARCHAR(20) UNIQUE NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium', -- low, medium, high, urgent
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- open, in_progress, waiting_customer, resolved, closed
    category VARCHAR(50), -- billing, technical, feature_request, bug_report
    assigned_to BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    closed_at TIMESTAMP WITH TIME ZONE,
    first_response_at TIMESTAMP WITH TIME ZONE,
    sla_due_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS ticket_comments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    is_staff BOOLEAN NOT NULL DEFAULT false,
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ticket_attachments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- RLS Policies
ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_attachments ENABLE ROW LEVEL SECURITY;

CREATE POLICY support_tickets_tenant_isolation ON support_tickets
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

CREATE POLICY ticket_comments_tenant_isolation ON ticket_comments
    USING (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

CREATE POLICY ticket_attachments_tenant_isolation ON ticket_attachments
    USING (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

-- Indexes
CREATE INDEX idx_support_tickets_tenant ON support_tickets(tenant_id);
CREATE INDEX idx_support_tickets_user ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_assigned ON support_tickets(assigned_to);
CREATE INDEX idx_ticket_comments_ticket ON ticket_comments(ticket_id);
CREATE INDEX idx_ticket_attachments_ticket ON ticket_attachments(ticket_id);

-- Function to generate ticket number
CREATE OR REPLACE FUNCTION generate_ticket_number()
RETURNS TRIGGER AS $$
BEGIN
    NEW.ticket_number := 'TKT-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(NEW.id::TEXT, 6, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_ticket_number
    AFTER INSERT ON support_tickets
    FOR EACH ROW
    EXECUTE FUNCTION generate_ticket_number();
```

**Requirements**: 37.1, 37.2, 37.3, 37.4

---

### 39.2 تطوير Ticket Manager ⏳

**File**: `services/backend/internal/support/ticket_manager.go`

```go
package support

import (
    "context"
    "database/sql"
    "time"
)

type Priority string
type Status string
type Category string

const (
    PriorityLow    Priority = "low"
    PriorityMedium Priority = "medium"
    PriorityHigh   Priority = "high"
    PriorityUrgent Priority = "urgent"
    
    StatusOpen            Status = "open"
    StatusInProgress      Status = "in_progress"
    StatusWaitingCustomer Status = "waiting_customer"
    StatusResolved        Status = "resolved"
    StatusClosed          Status = "closed"
    
    CategoryBilling        Category = "billing"
    CategoryTechnical      Category = "technical"
    CategoryFeatureRequest Category = "feature_request"
    CategoryBugReport      Category = "bug_report"
)

type Ticket struct {
    ID              int64      `json:"id"`
    TenantID        int64      `json:"tenant_id"`
    UserID          int64      `json:"user_id"`
    TicketNumber    string     `json:"ticket_number"`
    Subject         string     `json:"subject"`
    Description     string     `json:"description"`
    Priority        Priority   `json:"priority"`
    Status          Status     `json:"status"`
    Category        Category   `json:"category"`
    AssignedTo      *int64     `json:"assigned_to"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
    ResolvedAt      *time.Time `json:"resolved_at"`
    ClosedAt        *time.Time `json:"closed_at"`
    FirstResponseAt *time.Time `json:"first_response_at"`
    SLADueAt        *time.Time `json:"sla_due_at"`
}

type Comment struct {
    ID        int64     `json:"id"`
    TicketID  int64     `json:"ticket_id"`
    UserID    int64     `json:"user_id"`
    IsStaff   bool      `json:"is_staff"`
    Comment   string    `json:"comment"`
    CreatedAt time.Time `json:"created_at"`
}

type Attachment struct {
    ID         int64     `json:"id"`
    TicketID   int64     `json:"ticket_id"`
    Filename   string    `json:"filename"`
    FilePath   string    `json:"file_path"`
    FileSize   int64     `json:"file_size"`
    MimeType   string    `json:"mime_type"`
    UploadedBy int64     `json:"uploaded_by"`
    CreatedAt  time.Time `json:"created_at"`
}

type TicketManager struct {
    db *sql.DB
}

func NewTicketManager(db *sql.DB) *TicketManager {
    return &TicketManager{db: db}
}

// CreateTicket creates a new support ticket
func (m *TicketManager) CreateTicket(ctx context.Context, ticket *Ticket) error {
    // TODO: Implement
    // - Insert ticket
    // - Calculate SLA due time
    // - Send notification to support team
    // - Send confirmation to user
    return nil
}

// UpdateTicket updates ticket status/priority
func (m *TicketManager) UpdateTicket(ctx context.Context, ticketID int64, updates map[string]interface{}) error {
    // TODO: Implement
    // - Update ticket fields
    // - Log status changes
    // - Send notifications
    return nil
}

// CloseTicket closes a ticket
func (m *TicketManager) CloseTicket(ctx context.Context, ticketID int64) error {
    // TODO: Implement
    // - Update status to closed
    // - Set closed_at timestamp
    // - Send closure notification
    return nil
}

// AddComment adds a comment to a ticket
func (m *TicketManager) AddComment(ctx context.Context, comment *Comment) error {
    // TODO: Implement
    // - Insert comment
    // - Update ticket updated_at
    // - Set first_response_at if first staff comment
    // - Send notification
    return nil
}

// AddAttachment adds an attachment to a ticket
func (m *TicketManager) AddAttachment(ctx context.Context, attachment *Attachment) error {
    // TODO: Implement
    // - Save file to storage
    // - Insert attachment record
    // - Update ticket
    return nil
}

// GetTicket retrieves a ticket by ID
func (m *TicketManager) GetTicket(ctx context.Context, tenantID, ticketID int64) (*Ticket, error) {
    // TODO: Implement
    return nil, nil
}

// ListTickets lists tickets with filters
func (m *TicketManager) ListTickets(ctx context.Context, tenantID int64, filters map[string]interface{}) ([]Ticket, error) {
    // TODO: Implement
    return nil, nil
}
```

**Requirements**: 37.2, 37.6, 37.7

---

### 39.3 تطوير Notification System للتذاكر ⏳

**Integration with existing notification service**:

```go
// services/backend/internal/notifications/ticket_notifications.go
package notifications

// SendTicketCreated sends notification when ticket is created
func (s *Service) SendTicketCreated(ctx context.Context, ticket *support.Ticket) error {
    // TODO: Implement
    // - Send email to user (confirmation)
    // - Send email to support team (new ticket alert)
    return nil
}

// SendTicketUpdated sends notification when ticket is updated
func (s *Service) SendTicketUpdated(ctx context.Context, ticket *support.Ticket) error {
    // TODO: Implement
    return nil
}

// SendNewComment sends notification for new comment
func (s *Service) SendNewComment(ctx context.Context, ticket *support.Ticket, comment *support.Comment) error {
    // TODO: Implement
    return nil
}
```

**Requirements**: 37.5, 37.8

---

### 39.4 تطوير SLA Tracking ⏳

**File**: `services/backend/internal/support/sla_tracker.go`

```go
package support

import (
    "context"
    "time"
)

type SLAConfig struct {
    Priority         Priority
    FirstResponseSLA time.Duration // Time to first response
    ResolutionSLA    time.Duration // Time to resolution
}

var DefaultSLAConfig = map[Priority]SLAConfig{
    PriorityUrgent: {
        Priority:         PriorityUrgent,
        FirstResponseSLA: 1 * time.Hour,
        ResolutionSLA:    4 * time.Hours,
    },
    PriorityHigh: {
        Priority:         PriorityHigh,
        FirstResponseSLA: 4 * time.Hours,
        ResolutionSLA:    24 * time.Hours,
    },
    PriorityMedium: {
        Priority:         PriorityMedium,
        FirstResponseSLA: 8 * time.Hours,
        ResolutionSLA:    48 * time.Hours,
    },
    PriorityLow: {
        Priority:         PriorityLow,
        FirstResponseSLA: 24 * time.Hours,
        ResolutionSLA:    72 * time.Hours,
    },
}

type SLATracker struct {
    db *sql.DB
}

func NewSLATracker(db *sql.DB) *SLATracker {
    return &SLATracker{db: db}
}

// CalculateSLADueTime calculates when SLA is due
func (t *SLATracker) CalculateSLADueTime(ticket *Ticket) time.Time {
    config := DefaultSLAConfig[ticket.Priority]
    return ticket.CreatedAt.Add(config.ResolutionSLA)
}

// CheckSLABreach checks if ticket has breached SLA
func (t *SLATracker) CheckSLABreach(ctx context.Context, ticketID int64) (bool, error) {
    // TODO: Implement
    return false, nil
}

// SendSLAAlerts sends alerts for tickets approaching SLA breach
func (t *SLATracker) SendSLAAlerts(ctx context.Context) error {
    // TODO: Implement
    // - Find tickets approaching SLA (80% of time elapsed)
    // - Send alerts to assigned staff
    // - Escalate if needed
    return nil
}

// GetSLAMetrics gets SLA performance metrics
func (t *SLATracker) GetSLAMetrics(ctx context.Context, tenantID int64, from, to time.Time) (*SLAMetrics, error) {
    // TODO: Implement
    return nil, nil
}

type SLAMetrics struct {
    TotalTickets          int     `json:"total_tickets"`
    TicketsWithinSLA      int     `json:"tickets_within_sla"`
    TicketsBreachedSLA    int     `json:"tickets_breached_sla"`
    SLAComplianceRate     float64 `json:"sla_compliance_rate"`
    AvgFirstResponseTime  float64 `json:"avg_first_response_time"`
    AvgResolutionTime     float64 `json:"avg_resolution_time"`
}
```

**Requirements**: 37.9

---

### 39.5 إنشاء Knowledge Base ⏳

**Migration**: `services/backend/migrations/031_knowledge_base_table.sql`

```sql
CREATE TABLE IF NOT EXISTS knowledge_base_articles (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags TEXT[],
    author_id BIGINT NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false,
    view_count BIGINT NOT NULL DEFAULT 0,
    helpful_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_kb_articles_tenant ON knowledge_base_articles(tenant_id);
CREATE INDEX idx_kb_articles_category ON knowledge_base_articles(category);
CREATE INDEX idx_kb_articles_published ON knowledge_base_articles(published);
CREATE INDEX idx_kb_articles_slug ON knowledge_base_articles(slug);

-- Full text search
CREATE INDEX idx_kb_articles_search ON knowledge_base_articles 
    USING gin(to_tsvector('english', title || ' ' || content));
```

**Requirements**: 37.10

---

### 39.6 تطوير Priority Support للشركات ⏳

**Enhanced SLA for Enterprise Customers**:

```go
// services/backend/internal/support/enterprise_support.go
package support

var EnterpriseSLAConfig = map[Priority]SLAConfig{
    PriorityUrgent: {
        Priority:         PriorityUrgent,
        FirstResponseSLA: 15 * time.Minute, // 15 minutes for enterprise
        ResolutionSLA:    2 * time.Hours,   // 2 hours for enterprise
    },
    PriorityHigh: {
        Priority:         PriorityHigh,
        FirstResponseSLA: 1 * time.Hour,
        ResolutionSLA:    8 * time.Hours,
    },
    // ... other priorities
}

// IsEnterpriseCustomer checks if customer has enterprise support
func (m *TicketManager) IsEnterpriseCustomer(ctx context.Context, tenantID int64) (bool, error) {
    // TODO: Implement
    // - Check if tenant has dedicated gateway
    // - Check subscription level
    return false, nil
}

// GetSLAConfig returns appropriate SLA config based on customer type
func (m *TicketManager) GetSLAConfig(ctx context.Context, tenantID int64) (map[Priority]SLAConfig, error) {
    isEnterprise, err := m.IsEnterpriseCustomer(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    
    if isEnterprise {
        return EnterpriseSLAConfig, nil
    }
    
    return DefaultSLAConfig, nil
}
```

**Requirements**: 37.12

---

## المهمة 40: وضع الاختبار (Test Mode)

### 40.1 تطوير Test Mode Environment ⏳

**File**: `services/backend/internal/testmode/manager.go`

```go
package testmode

import (
    "context"
    "strings"
)

type Manager struct {
    enabled bool
}

func NewManager(enabled bool) *Manager {
    return &Manager{enabled: enabled}
}

// IsTestMode checks if system is in test mode
func (m *Manager) IsTestMode(ctx context.Context) bool {
    // Check from context or API key
    apiKey := ctx.Value("api_key").(string)
    return m.enabled || strings.HasPrefix(apiKey, "test_")
}

// MockSendMessage mocks message sending in test mode
func (m *Manager) MockSendMessage(ctx context.Context, message *Message) (*MessageResult, error) {
    if !m.IsTestMode(ctx) {
        return nil, errors.New("not in test mode")
    }
    
    // Return mock result based on phone number
    result := &MessageResult{
        MessageID: "test_" + generateID(),
        Status:    m.getTestStatus(message.To),
        Timestamp: time.Now(),
    }
    
    return result, nil
}

// getTestStatus returns status based on test phone number
func (m *Manager) getTestStatus(phoneNumber string) string {
    // Test numbers:
    // +967777000001 -> success
    // +967777000002 -> failed
    // +967777000003 -> pending
    
    switch phoneNumber {
    case "+967777000001":
        return "delivered"
    case "+967777000002":
        return "failed"
    case "+967777000003":
        return "pending"
    default:
        return "delivered"
    }
}
```

**Requirements**: 38.1, 38.2, 38.3, 38.4

---

### 40.2 تطوير Test Phone Numbers ⏳

**Configuration**:

```go
// services/backend/internal/testmode/test_numbers.go
package testmode

var TestPhoneNumbers = map[string]string{
    "+967777000001": "success",
    "+967777000002": "failed",
    "+967777000003": "pending",
    "+967777000004": "invalid_number",
    "+967777000005": "insufficient_balance",
    "+967777000006": "rate_limited",
}

// IsTestPhoneNumber checks if phone number is a test number
func IsTestPhoneNumber(phoneNumber string) bool {
    _, exists := TestPhoneNumbers[phoneNumber]
    return exists
}

// GetTestResult returns the expected result for a test number
func GetTestResult(phoneNumber string) string {
    if result, exists := TestPhoneNumbers[phoneNumber]; exists {
        return result
    }
    return "success" // default
}
```

**Requirements**: 38.5, 38.6

---

### 40.3 تطوير Test API Keys ⏳

**API Key Format**:
- Production: `pk_live_...`
- Test: `pk_test_...`

```go
// services/backend/internal/auth/api_key.go

// IsTestAPIKey checks if API key is for test mode
func IsTestAPIKey(apiKey string) bool {
    return strings.HasPrefix(apiKey, "pk_test_") || strings.HasPrefix(apiKey, "sk_test_")
}

// ValidateAPIKey validates API key and returns mode
func ValidateAPIKey(ctx context.Context, apiKey string) (bool, string, error) {
    // Validate key format
    if !isValidKeyFormat(apiKey) {
        return false, "", errors.New("invalid API key format")
    }
    
    // Determine mode
    mode := "live"
    if IsTestAPIKey(apiKey) {
        mode = "test"
    }
    
    // Validate key in database
    valid, err := validateKeyInDB(ctx, apiKey)
    if err != nil {
        return false, "", err
    }
    
    return valid, mode, nil
}
```

**Requirements**: 38.7, 38.8, 38.9

---

### 40.4 تطوير Test Logging ⏳

**Separate logging for test mode**:

```go
// services/backend/internal/testmode/logger.go
package testmode

type TestLogger struct {
    db *sql.DB
}

func NewTestLogger(db *sql.DB) *TestLogger {
    return &TestLogger{db: db}
}

// LogTestOperation logs test mode operations
func (l *TestLogger) LogTestOperation(ctx context.Context, operation *TestOperation) error {
    // TODO: Implement
    // - Log to separate test_operations table
    // - Don't affect production statistics
    // - Include test mode indicator
    return nil
}

type TestOperation struct {
    ID          int64     `json:"id"`
    TenantID    int64     `json:"tenant_id"`
    APIKey      string    `json:"api_key"`
    Operation   string    `json:"operation"`
    TestNumber  string    `json:"test_number"`
    Result      string    `json:"result"`
    Timestamp   time.Time `json:"timestamp"`
}
```

**Migration**:
```sql
CREATE TABLE IF NOT EXISTS test_operations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    api_key VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    test_number VARCHAR(20),
    result VARCHAR(50),
    request_data JSONB,
    response_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_test_operations_tenant ON test_operations(tenant_id);
CREATE INDEX idx_test_operations_created ON test_operations(created_at);
```

**Requirements**: 38.10

---

## خطة التنفيذ

### الأولوية 1: نظام الدعم الأساسي
1. ✅ إنشاء جداول support_tickets
2. ✅ تطوير Ticket Manager
3. ✅ تطوير Notification System
4. ⏳ اختبار النظام

### الأولوية 2: SLA و Knowledge Base
1. ⏳ تطوير SLA Tracking
2. ⏳ إنشاء Knowledge Base
3. ⏳ تطوير Priority Support

### الأولوية 3: Test Mode
1. ⏳ تطوير Test Mode Manager
2. ⏳ تطوير Test Phone Numbers
3. ⏳ تطوير Test API Keys
4. ⏳ تطوير Test Logging

## الوقت المقدر

- نظام الدعم الأساسي: 1 أسبوع
- SLA و Knowledge Base: 3-4 أيام
- Priority Support: 2-3 أيام
- Test Mode: 3-4 أيام

**الإجمالي**: 2-3 أسابيع

## الخلاصة

المهام 39 و 40 هي مهام اختيارية لكنها مفيدة:
- ✅ الإطار التقني جاهز
- ⏳ التنفيذ يمكن أن يكون تدريجياً
- ⏳ يمكن البدء بالوظائف الأساسية أولاً

**التوصية**: تنفيذ نظام الدعم الأساسي أولاً، ثم إضافة المزايا المتقدمة تدريجياً.

---

**Status**: 📋 Framework Ready - Can Be Implemented Incrementally
**Date**: 2026-03-02
**Requirements**: 37.1-37.12, 38.1-38.10
