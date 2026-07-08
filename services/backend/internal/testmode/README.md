# Test Mode System

نظام شامل لاختبار الرسائل دون إرسال رسائل حقيقية أو خصم رصيد.

## نظرة عامة

Test Mode يوفر بيئة آمنة لاختبار التكامل مع النظام دون:
- إرسال رسائل حقيقية
- خصم رصيد من الحساب
- التأثير على الإحصائيات الحقيقية
- استهلاك الموارد الفعلية

## المكونات

### 1. Test Mode Manager (`manager.go`)
إدارة وضع الاختبار:
- التحقق من وضع الاختبار
- محاكاة إرسال الرسائل
- تسجيل العمليات
- إحصائيات الاختبار

### 2. Test Phone Numbers (`test_numbers.go`)
أرقام اختبار محددة مسبقاً:
- أرقام للنجاح
- أرقام للفشل
- أرقام للحالات المعلقة
- أرقام لأخطاء مختلفة

### 3. Test API Keys (`api_keys.go`)
مفاتيح API للاختبار:
- توليد مفاتيح اختبار
- التحقق من المفاتيح
- إخفاء المفاتيح للعرض

### 4. Test Logging
تسجيل منفصل للعمليات:
- سجل مستقل عن الإنتاج
- لا يؤثر على الإحصائيات
- سهولة التنظيف

## الاستخدام

### تفعيل Test Mode

#### 1. استخدام Test API Key

```go
// Create test API key
testKey, err := testmode.GenerateTestAPIKey(true) // public key
// Result: pk_test_abc123...

// Use in requests
ctx := context.WithValue(context.Background(), "api_key", testKey)
```

#### 2. التحقق من Test Mode

```go
manager := testmode.NewManager(db, false)

if manager.IsTestMode(ctx) {
    fmt.Println("Running in test mode")
}
```

### إرسال رسالة اختبار

```go
message := &testmode.Message{
    To:       "+967777000001", // Test number for success
    From:     "TestSender",
    Body:     "This is a test message",
    TenantID: 1,
}

result, err := manager.MockSendMessage(ctx, message)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message ID: %s\n", result.MessageID)
fmt.Printf("Status: %s\n", result.Status)
```

### الحصول على إحصائيات الاختبار

```go
stats, err := manager.GetTestStatistics(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total Operations: %d\n", stats.TotalOperations)
fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate)
```

## Test Phone Numbers

### أرقام النجاح
```
+967777000001  → delivered
+967777000011  → delivered
+967777000021  → delivered
```

### أرقام الفشل
```
+967777000002  → failed
+967777000012  → failed
+967777000022  → failed
```

### أرقام معلقة
```
+967777000003  → pending
+967777000013  → pending
+967777000023  → pending
```

### أرقام الأخطاء
```
+967777000004  → invalid_number
+967777000005  → insufficient_balance
+967777000006  → rate_limited
```

### أرقام دولية
```
+1555000001    → delivered (US)
+44555000001   → delivered (UK)
+966555000001  → delivered (Saudi Arabia)
+971555000001  → delivered (UAE)
+20555000001   → delivered (Egypt)
```

## API Keys

### أنواع المفاتيح

#### Test Keys (للاختبار)
```
pk_test_[64 hex chars]  - Public Test Key
sk_test_[64 hex chars]  - Secret Test Key
```

#### Live Keys (للإنتاج)
```
pk_live_[64 hex chars]  - Public Live Key
sk_live_[64 hex chars]  - Secret Live Key
```

### توليد مفاتيح جديدة

```go
// Generate test public key
publicKey, err := testmode.GenerateTestAPIKey(true)

// Generate test secret key
secretKey, err := testmode.GenerateTestAPIKey(false)

// Generate live keys
livePublicKey, err := testmode.GenerateLiveAPIKey(true)
liveSecretKey, err := testmode.GenerateLiveAPIKey(false)
```

### التحقق من المفاتيح

```go
// Check if test key
isTest := testmode.IsTestAPIKey(apiKey)

// Check if live key
isLive := testmode.IsLiveAPIKey(apiKey)

// Get key info
info := testmode.GetAPIKeyInfo(apiKey)
fmt.Printf("Mode: %s, Type: %s\n", info.Mode, info.Type)
```

### إخفاء المفاتيح

```go
masked := testmode.MaskAPIKey("pk_test_abc123def456...")
// Result: pk_test_abc1****f456
```

## Test Logging

### عرض سجل العمليات

```go
operations, err := manager.GetTestOperations(ctx, tenantID, 50, 0)
for _, op := range operations {
    fmt.Printf("%s: %s → %s\n", op.Operation, op.TestNumber, op.Result)
}
```

### تنظيف السجل

```go
err := manager.ClearTestOperations(ctx, tenantID)
```

## أمثلة كاملة

### مثال 1: اختبار إرسال ناجح

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "your-project/internal/testmode"
)

func main() {
    // Setup
    manager := testmode.NewManager(db, false)
    testKey, _ := testmode.GenerateTestAPIKey(true)
    ctx := context.WithValue(context.Background(), "api_key", testKey)
    
    // Send test message
    message := &testmode.Message{
        To:       "+967777000001", // Success number
        From:     "TestApp",
        Body:     "Hello from test mode!",
        TenantID: 1,
    }
    
    result, err := manager.MockSendMessage(ctx, message)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("✓ Message sent successfully\n")
    fmt.Printf("  Message ID: %s\n", result.MessageID)
    fmt.Printf("  Status: %s\n", result.Status)
}
```

### مثال 2: اختبار فشل الإرسال

```go
message := &testmode.Message{
    To:       "+967777000002", // Failure number
    From:     "TestApp",
    Body:     "This will fail",
    TenantID: 1,
}

result, err := manager.MockSendMessage(ctx, message)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("✗ Message failed as expected\n")
fmt.Printf("  Status: %s\n", result.Status)
fmt.Printf("  Error: %s\n", result.Error)
```

### مثال 3: اختبار رصيد غير كافي

```go
message := &testmode.Message{
    To:       "+967777000005", // Insufficient balance
    From:     "TestApp",
    Body:     "Testing balance check",
    TenantID: 1,
}

result, err := manager.MockSendMessage(ctx, message)
// Result will show insufficient_balance error
```

## Integration with Main System

### في Gateway Handler

```go
func (h *Handler) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
    // Check if in test mode
    if h.testManager.IsTestMode(ctx) {
        // Use test mode
        message := &testmode.Message{
            To:       req.To,
            From:     req.From,
            Body:     req.Body,
            TenantID: req.TenantID,
        }
        
        result, err := h.testManager.MockSendMessage(ctx, message)
        if err != nil {
            return nil, err
        }
        
        return &SendMessageResponse{
            MessageID: result.MessageID,
            Status:    result.Status,
        }, nil
    }
    
    // Use real sending logic
    return h.sendRealMessage(ctx, req)
}
```

## Best Practices

### 1. استخدام Test Mode في التطوير
```go
// Development environment
testManager := testmode.NewManager(db, true) // Always test mode

// Production environment
testManager := testmode.NewManager(db, false) // Based on API key
```

### 2. التحقق من الأرقام
```go
if !testmode.IsTestPhoneNumber(phoneNumber) && manager.IsTestMode(ctx) {
    return errors.New("use test phone numbers in test mode")
}
```

### 3. تنظيف السجلات دورياً
```go
// Cleanup old test operations (30+ days)
_, err := db.Exec("SELECT cleanup_old_test_operations()")
```

### 4. مراقبة الاستخدام
```go
stats, _ := manager.GetTestStatistics(ctx, tenantID)
if stats.TotalOperations > 10000 {
    log.Warn("High test mode usage detected")
}
```

## Database Schema

### test_operations Table
```sql
CREATE TABLE test_operations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    api_key VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    test_number VARCHAR(20),
    result VARCHAR(50),
    request_data TEXT,
    response_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Requirements Covered

- ✅ 38.1: Test mode environment
- ✅ 38.2: No real messages sent
- ✅ 38.3: No balance deducted
- ✅ 38.4: Mock all operations
- ✅ 38.5: Test phone numbers
- ✅ 38.6: Different test results
- ✅ 38.7: Clear distinction
- ✅ 38.8: Easy switching
- ✅ 38.9: Separate test API keys
- ✅ 38.10: Separate logging

## Testing

```bash
# Run tests
go test ./internal/testmode/...

# Run with coverage
go test -cover ./internal/testmode/...

# Run specific test
go test -run TestMockSendMessage ./internal/testmode/...
```

## Troubleshooting

### Problem: Test mode not activating
**Solution**: Verify API key has `pk_test_` or `sk_test_` prefix

### Problem: Test number not recognized
**Solution**: Use numbers from the predefined list (+967777000xxx)

### Problem: Operations not logged
**Solution**: Check database connection and RLS policies

### Problem: Cannot switch between modes
**Solution**: Use different API keys for test and live modes

## Future Enhancements

- [ ] Webhook simulation
- [ ] Delay simulation
- [ ] Network error simulation
- [ ] Batch testing support
- [ ] Test scenarios builder
- [ ] Performance testing mode
- [ ] Test data generator
- [ ] Visual test dashboard

## Documentation

For more information:
- Test Numbers: `testmode.GetTestNumbersDocumentation()`
- API Keys: `testmode.GetAPIKeyDocumentation()`
- All Test Numbers: `testmode.GetAllTestNumbers()`

## Support

For issues or questions about test mode:
1. Check this README
2. Review test examples
3. Check test operation logs
4. Contact development team
