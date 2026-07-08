# Support Ticket System

نظام شامل لإدارة تذاكر الدعم الفني مع دعم SLA وقاعدة المعرفة.

## المكونات

### 1. Ticket Manager (`ticket_manager.go`)
إدارة دورة حياة التذاكر الكاملة:
- إنشاء تذاكر جديدة
- تحديث حالة التذاكر
- إضافة تعليقات ومرفقات
- إغلاق التذاكر

### 2. SLA Tracker (`sla_tracker.go`)
تتبع ومراقبة SLA:
- حساب مواعيد SLA
- مراقبة الانتهاكات
- إرسال تنبيهات
- إحصائيات الأداء

### 3. Notification Service (`notifications.go`)
إرسال الإشعارات:
- إشعارات إنشاء التذاكر
- إشعارات التحديثات
- إشعارات التعليقات
- تنبيهات SLA

### 4. Knowledge Base
قاعدة معرفة للأسئلة الشائعة:
- مقالات مساعدة
- بحث نصي كامل
- تقييم المقالات

## الاستخدام

### إنشاء تذكرة جديدة

```go
manager := support.NewTicketManager(db)

req := &support.CreateTicketRequest{
    Subject:     "Cannot send messages",
    Description: "I'm getting an error when trying to send SMS",
    Priority:    support.PriorityHigh,
    Category:    support.CategoryTechnical,
}

ticket, err := manager.CreateTicket(ctx, tenantID, userID, req)
if err != nil {
    log.Fatal(err)
}
```

### إضافة تعليق

```go
commentReq := &support.AddCommentRequest{
    Comment: "We are investigating this issue",
}

comment, err := manager.AddComment(ctx, tenantID, ticketID, staffID, true, commentReq)
if err != nil {
    log.Fatal(err)
}
```

### تحديث حالة التذكرة

```go
updateReq := &support.UpdateTicketRequest{
    Status: ptrStatus(support.StatusResolved),
}

err := manager.UpdateTicket(ctx, tenantID, ticketID, updateReq)
if err != nil {
    log.Fatal(err)
}
```

### مراقبة SLA

```go
tracker := support.NewSLATracker(db, notificationSvc)

// Check for SLA breaches
err := tracker.CheckSLABreaches(ctx)
if err != nil {
    log.Fatal(err)
}

// Get SLA metrics
metrics, err := tracker.GetSLAMetrics(ctx, tenantID, from, to)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("SLA Compliance Rate: %.2f%%\n", metrics.SLAComplianceRate)
```

## SLA Configuration

### Standard Customers

| Priority | First Response | Resolution |
|----------|---------------|------------|
| Urgent   | 1 hour        | 4 hours    |
| High     | 4 hours       | 24 hours   |
| Medium   | 8 hours       | 48 hours   |
| Low      | 24 hours      | 72 hours   |

### Enterprise Customers

| Priority | First Response | Resolution |
|----------|---------------|------------|
| Urgent   | 15 minutes    | 2 hours    |
| High     | 1 hour        | 8 hours    |
| Medium   | 4 hours       | 24 hours   |
| Low      | 8 hours       | 48 hours   |

## Ticket Status Flow

```
open → in_progress → waiting_customer → resolved → closed
  ↓         ↓              ↓               ↓
  └─────────┴──────────────┴───────────────┘
              (can reopen)
```

## Priority Levels

- **Urgent**: Critical issues affecting service
- **High**: Important issues requiring quick resolution
- **Medium**: Standard issues
- **Low**: Minor issues or questions

## Categories

- **billing**: Billing and payment issues
- **technical**: Technical problems
- **feature_request**: Feature requests
- **bug_report**: Bug reports

## Database Schema

### support_tickets
- Stores all support tickets
- RLS enabled for tenant isolation
- Automatic ticket number generation

### ticket_comments
- Stores comments on tickets
- Tracks staff vs customer comments
- Updates first_response_at automatically

### ticket_attachments
- Stores file attachments
- Supports multiple files per ticket

### knowledge_base_articles
- Stores help articles
- Full-text search enabled
- View and helpful counts

## Notifications

The system sends notifications for:
- Ticket creation
- Status updates
- New comments
- SLA alerts
- SLA breaches
- Ticket assignment

## Enterprise Support

Enterprise customers (those with dedicated gateways) receive:
- Faster SLA times
- Priority routing
- Dedicated support staff
- Enhanced monitoring

## Requirements Covered

- ✅ 37.1: Support ticket system
- ✅ 37.2: Create tickets
- ✅ 37.3: Priority classification
- ✅ 37.4: Type classification
- ✅ 37.5: Creation notifications
- ✅ 37.6: Status tracking
- ✅ 37.7: Comments and attachments
- ✅ 37.8: Update notifications
- ✅ 37.9: SLA tracking
- ✅ 37.10: Knowledge base
- ✅ 37.11: User guide
- ✅ 37.12: Priority support for enterprises

## Testing

```bash
# Run tests
go test ./internal/support/...

# Run with coverage
go test -cover ./internal/support/...
```

## Future Enhancements

- [ ] Email integration for ticket creation
- [ ] Automatic ticket assignment
- [ ] Ticket templates
- [ ] Canned responses
- [ ] Customer satisfaction surveys
- [ ] Advanced analytics
- [ ] Multi-language support
- [ ] Live chat integration

## Notes

- All operations respect tenant isolation via RLS
- SLA calculations exclude weekends and holidays (future enhancement)
- Notifications are logged but not sent in development mode
- Enterprise status is determined by presence of dedicated gateway
