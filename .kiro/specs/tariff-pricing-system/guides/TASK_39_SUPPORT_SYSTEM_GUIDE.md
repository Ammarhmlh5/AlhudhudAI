# دليل نظام الدعم الفني - المهمة 39

## نظرة عامة

دليل شامل لتطوير نظام الدعم الفني للعملاء.

## المهمة 39.1: Support Tickets System

### إنشاء جدول التذاكر

```sql
-- services/backend/migrations/029_support_tickets_table.sql
CREATE TABLE support_tickets (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal', -- low, normal, high, urgent
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- open, in_progress, resolved, closed
    category VARCHAR(50), -- billing, technical, general
    assigned_to UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    
    CONSTRAINT check_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT check_status CHECK (status IN ('open', 'in_progress', 'resolved', 'closed'))
);

CREATE INDEX idx_support_tickets_tenant ON support_tickets(tenant_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_assigned ON support_tickets(assigned_to);

-- تفعيل RLS
ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON support_tickets
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### جدول التعليقات والمرفقات

```sql
CREATE TABLE ticket_comments (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL REFERENCES support_tickets(id),
    user_id UUID NOT NULL,
    comment TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ticket_attachments (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL REFERENCES support_tickets(id),
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100),
    uploaded_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## المهمة 39.2: Ticket Manager

### تطوير مدير التذاكر

```go
// services/backend/internal/support/ticket_manager.go
package support

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type TicketManager struct {
    db *gorm.DB
}

func NewTicketManager(db *gorm.DB) *TicketManager {
    return &TicketManager{db: db}
}

func (tm *TicketManager) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*Ticket, error) {
    ticket := &Ticket{
        TenantID:    req.TenantID,
        UserID:      req.UserID,
        Subject:     req.Subject,
        Description: req.Description,
        Priority:    req.Priority,
        Category:    req.Category,
        Status:      "open",
    }
    
    // تحديد الأولوية تلقائياً للشركات المؤسسية
    if tm.isEnterpriseTenant(req.TenantID) {
        ticket.Priority = "high"
    }
    
    if err := tm.db.Create(ticket).Error; err != nil {
        return nil, err
    }
    
    // إرسال إشعار للدعم
    tm.notifySupport(ticket)
    
    return ticket, nil
}

func (tm *TicketManager) UpdateTicket(ctx context.Context, ticketID int, updates map[string]interface{}) error {
    return tm.db.Model(&Ticket{}).
        Where("id = ?", ticketID).
        Updates(updates).Error
}

func (tm *TicketManager) CloseTicket(ctx context.Context, ticketID int) error {
    return tm.db.Model(&Ticket{}).
        Where("id = ?", ticketID).
        Updates(map[string]interface{}{
            "status":    "closed",
            "closed_at": time.Now(),
        }).Error
}

func (tm *TicketManager) AddComment(ctx context.Context, ticketID int, userID uuid.UUID, comment string, isInternal bool) error {
    tc := &TicketComment{
        TicketID:   ticketID,
        UserID:     userID,
        Comment:    comment,
        IsInternal: isInternal,
    }
    
    if err := tm.db.Create(tc).Error; err != nil {
        return err
    }
    
    // تحديث updated_at للتذكرة
    tm.db.Model(&Ticket{}).Where("id = ?", ticketID).Update("updated_at", time.Now())
    
    // إرسال إشعار
    if !isInternal {
        tm.notifyTicketOwner(ticketID, comment)
    }
    
    return nil
}

func (tm *TicketManager) AddAttachment(ctx context.Context, ticketID int, filename, filePath string, fileSize int, mimeType string, uploadedBy uuid.UUID) error {
    attachment := &TicketAttachment{
        TicketID:   ticketID,
        Filename:   filename,
        FilePath:   filePath,
        FileSize:   fileSize,
        MimeType:   mimeType,
        UploadedBy: uploadedBy,
    }
    
    return tm.db.Create(attachment).Error
}
```

## المهمة 39.3: Notification System

### إشعارات التذاكر

```go
func (tm *TicketManager) notifySupport(ticket *Ticket) {
    // إرسال بريد إلكتروني للدعم
    subject := fmt.Sprintf("[Ticket #%d] %s", ticket.ID, ticket.Subject)
    body := fmt.Sprintf(`
        تذكرة جديدة من العميل:
        
        الموضوع: %s
        الأولوية: %s
        الفئة: %s
        
        الوصف:
        %s
        
        رابط التذكرة: https://admin.example.com/tickets/%d
    `, ticket.Subject, ticket.Priority, ticket.Category, ticket.Description, ticket.ID)
    
    tm.sendEmail("support@example.com", subject, body)
}

func (tm *TicketManager) notifyTicketOwner(ticketID int, comment string) {
    var ticket Ticket
    tm.db.First(&ticket, ticketID)
    
    // إرسال إشعار للعميل
    subject := fmt.Sprintf("رد جديد على تذكرتك #%d", ticketID)
    body := fmt.Sprintf(`
        تم إضافة رد جديد على تذكرتك:
        
        %s
        
        رابط التذكرة: https://app.example.com/tickets/%d
    `, comment, ticketID)
    
    tm.sendEmailToTenant(ticket.TenantID, subject, body)
}
```

## المهمة 39.4: SLA Tracking

### تتبع SLA

```go
type SLATracker struct {
    db *gorm.DB
}

func (st *SLATracker) CalculateSLA(ticket *Ticket) *SLAMetrics {
    // تحديد SLA حسب الأولوية ونوع العميل
    var responseTime, resolutionTime time.Duration
    
    isEnterprise := st.isEnterpriseTenant(ticket.TenantID)
    
    switch ticket.Priority {
    case "urgent":
        if isEnterprise {
            responseTime = 15 * time.Minute
            resolutionTime = 2 * time.Hour
        } else {
            responseTime = 30 * time.Minute
            resolutionTime = 4 * time.Hour
        }
    case "high":
        if isEnterprise {
            responseTime = 1 * time.Hour
            resolutionTime = 8 * time.Hour
        } else {
            responseTime = 2 * time.Hour
            resolutionTime = 24 * time.Hour
        }
    case "normal":
        responseTime = 4 * time.Hour
        resolutionTime = 48 * time.Hour
    case "low":
        responseTime = 24 * time.Hour
        resolutionTime = 7 * 24 * time.Hour
    }
    
    // حساب الوقت الفعلي
    var firstResponse time.Time
    st.db.Model(&TicketComment{}).
        Where("ticket_id = ? AND is_internal = false", ticket.ID).
        Order("created_at ASC").
        Limit(1).
        Pluck("created_at", &firstResponse)
    
    actualResponseTime := firstResponse.Sub(ticket.CreatedAt)
    
    var actualResolutionTime time.Duration
    if ticket.ResolvedAt != nil {
        actualResolutionTime = ticket.ResolvedAt.Sub(ticket.CreatedAt)
    }
    
    return &SLAMetrics{
        TicketID:              ticket.ID,
        ExpectedResponseTime:  responseTime,
        ActualResponseTime:    actualResponseTime,
        ResponseSLAMet:        actualResponseTime <= responseTime,
        ExpectedResolutionTime: resolutionTime,
        ActualResolutionTime:   actualResolutionTime,
        ResolutionSLAMet:       actualResolutionTime <= resolutionTime,
    }
}

func (st *SLATracker) CheckSLAViolations() {
    var tickets []Ticket
    st.db.Where("status IN ('open', 'in_progress')").Find(&tickets)
    
    for _, ticket := range tickets {
        sla := st.CalculateSLA(&ticket)
        
        if !sla.ResponseSLAMet || !sla.ResolutionSLAMet {
            // إرسال تنبيه
            st.alertSLAViolation(&ticket, sla)
        }
    }
}
```

## المهمة 39.5: Knowledge Base

### قاعدة المعرفة

```sql
CREATE TABLE knowledge_base (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(50),
    tags TEXT[],
    views INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    is_published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_kb_category ON knowledge_base(category);
CREATE INDEX idx_kb_published ON knowledge_base(is_published);
CREATE INDEX idx_kb_tags ON knowledge_base USING GIN(tags);
```

### مقالات شائعة

```markdown
# مقالات قاعدة المعرفة

## 1. كيفية الاشتراك في باقة
- اذهب إلى صفحة الباقات
- اختر الباقة المناسبة
- انقر على "اشترك الآن"
- تأكد من كفاية رصيدك

## 2. ما هو نظام الترحيل (Rollover)؟
الترحيل يسمح لك بنقل الرسائل المتبقية للشهر التالي...

## 3. كيفية شحن الرصيد
- اذهب إلى "الرصيد"
- انقر على "شحن الرصيد"
- اختر طريقة الدفع
- أكمل عملية الدفع

## 4. ماذا يحدث عند نفاد الرسائل؟
عند نفاد رسائلك المضمنة، يتم تفعيل Overage تلقائياً...
```

## المهمة 39.6: Priority Support

### دعم أولوية للشركات

```go
func (tm *TicketManager) AssignTicket(ticket *Ticket) {
    // الشركات المؤسسية تحصل على أولوية
    if tm.isEnterpriseTenant(ticket.TenantID) {
        // تعيين لأفضل مهندس دعم متاح
        var supportAgent User
        tm.db.Where("role = 'senior_support' AND is_available = true").
            Order("current_tickets ASC").
            First(&supportAgent)
        
        ticket.AssignedTo = &supportAgent.ID
        ticket.Priority = "high"
        tm.db.Save(ticket)
        
        // إشعار فوري
        tm.notifySupportAgent(supportAgent.ID, ticket)
    } else {
        // تعيين عادي
        tm.autoAssignTicket(ticket)
    }
}
```

## Checklist للتحقق

- [ ] نظام التذاكر يعمل بشكل كامل
- [ ] الإشعارات تُرسل بشكل صحيح
- [ ] SLA Tracking نشط
- [ ] Knowledge Base متاحة
- [ ] Priority Support للشركات المؤسسية

**Requirements Covered:** 37.1-37.12
