# دليل تنفيذ الامتثال القانوني - نظام التعرفات والتسعير

## نظرة عامة

هذا الدليل يوفر إطار عمل لتنفيذ متطلبات الامتثال القانوني لنظام التعرفات والتسعير. المهام 38.1-38.7 تتطلب مراجعة قانونية ومحتوى متخصص.

## ⚠️ تنبيه مهم

**المحتوى القانوني يتطلب**:
- مراجعة من مستشار قانوني مؤهل
- تخصيص حسب القوانين المحلية (اليمن، السعودية، إلخ)
- تحديث دوري حسب التغييرات التشريعية
- ترجمة معتمدة للغات المطلوبة

## المهام المتبقية

### 38.1 إنشاء صفحة شروط الخدمة ⏳

**الملف**: `apps/client-web/src/pages/TermsOfService.tsx`

**المحتوى المطلوب**:
1. تعريف الخدمة
2. حقوق والتزامات المستخدم
3. سياسة الدفع والاسترداد
4. حدود المسؤولية
5. إنهاء الخدمة
6. القانون الحاكم
7. حل النزاعات

**قالب مقترح**:
```typescript
// apps/client-web/src/pages/TermsOfService.tsx
import React from 'react';
import { useTranslation } from 'react-i18n';

export const TermsOfService: React.FC = () => {
  const { t } = useTranslation();
  
  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">
        {t('legal.terms.title')}
      </h1>
      
      <section className="mb-8">
        <h2 className="text-2xl font-semibold mb-4">
          {t('legal.terms.section1.title')}
        </h2>
        <p className="text-gray-700">
          {t('legal.terms.section1.content')}
        </p>
      </section>
      
      {/* Add more sections */}
      
      <div className="mt-8 p-4 bg-gray-100 rounded">
        <p className="text-sm text-gray-600">
          {t('legal.terms.lastUpdated')}: {new Date().toLocaleDateString()}
        </p>
      </div>
    </div>
  );
};
```

**ملفات الترجمة**:
```json
// apps/client-web/src/i18n/locales/ar.json
{
  "legal": {
    "terms": {
      "title": "شروط الخدمة",
      "section1": {
        "title": "1. تعريف الخدمة",
        "content": "[يتطلب محتوى قانوني من مستشار]"
      }
    }
  }
}
```

**Requirements**: 36.1

---

### 38.2 إنشاء صفحة سياسة الخصوصية ⏳

**الملف**: `apps/client-web/src/pages/PrivacyPolicy.tsx`

**المحتوى المطلوب** (GDPR/CCPA Compliant):
1. البيانات المجمعة
2. كيفية استخدام البيانات
3. مشاركة البيانات مع أطراف ثالثة
4. حقوق المستخدم
5. أمن البيانات
6. ملفات تعريف الارتباط (Cookies)
7. التغييرات على السياسة
8. معلومات الاتصال

**قالب مقترح**:
```typescript
// apps/client-web/src/pages/PrivacyPolicy.tsx
import React from 'react';
import { useTranslation } from 'react-i18n';

export const PrivacyPolicy: React.FC = () => {
  const { t } = useTranslation();
  
  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">
        {t('legal.privacy.title')}
      </h1>
      
      <div className="prose max-w-none">
        {/* Data Collection */}
        <section className="mb-8">
          <h2>{t('legal.privacy.dataCollection.title')}</h2>
          <ul>
            <li>{t('legal.privacy.dataCollection.item1')}</li>
            <li>{t('legal.privacy.dataCollection.item2')}</li>
          </ul>
        </section>
        
        {/* User Rights */}
        <section className="mb-8">
          <h2>{t('legal.privacy.userRights.title')}</h2>
          <p>{t('legal.privacy.userRights.content')}</p>
        </section>
      </div>
    </div>
  );
};
```

**Requirements**: 36.2

---

### 38.3 تطوير Consent Management ⏳

**الملف**: `services/backend/internal/consent/manager.go`

**الوظائف المطلوبة**:
```go
package consent

import (
    "context"
    "time"
)

type ConsentType string

const (
    ConsentTermsOfService ConsentType = "terms_of_service"
    ConsentPrivacyPolicy  ConsentType = "privacy_policy"
    ConsentMarketing      ConsentType = "marketing"
    ConsentDataProcessing ConsentType = "data_processing"
)

type Consent struct {
    ID          int64       `json:"id"`
    TenantID    int64       `json:"tenant_id"`
    UserID      int64       `json:"user_id"`
    Type        ConsentType `json:"type"`
    Version     string      `json:"version"`
    Granted     bool        `json:"granted"`
    GrantedAt   time.Time   `json:"granted_at"`
    IPAddress   string      `json:"ip_address"`
    UserAgent   string      `json:"user_agent"`
}

type Manager struct {
    db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
    return &Manager{db: db}
}

// RecordConsent records user consent
func (m *Manager) RecordConsent(ctx context.Context, consent *Consent) error {
    // TODO: Implement
    // - Insert consent record
    // - Log audit trail
    // - Return error if failed
    return nil
}

// GetUserConsents retrieves all consents for a user
func (m *Manager) GetUserConsents(ctx context.Context, tenantID, userID int64) ([]Consent, error) {
    // TODO: Implement
    return nil, nil
}

// RevokeConsent revokes a specific consent
func (m *Manager) RevokeConsent(ctx context.Context, tenantID, userID int64, consentType ConsentType) error {
    // TODO: Implement
    return nil
}

// CheckConsent checks if user has granted specific consent
func (m *Manager) CheckConsent(ctx context.Context, tenantID, userID int64, consentType ConsentType) (bool, error) {
    // TODO: Implement
    return false, nil
}
```

**Migration**:
```sql
-- services/backend/migrations/029_user_consents_table.sql
CREATE TABLE IF NOT EXISTS user_consents (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    consent_type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- RLS Policies
ALTER TABLE user_consents ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_consents_tenant_isolation ON user_consents
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

-- Indexes
CREATE INDEX idx_user_consents_tenant_user ON user_consents(tenant_id, user_id);
CREATE INDEX idx_user_consents_type ON user_consents(consent_type);
```

**Requirements**: 36.3, 36.4

---

### 38.4 تطوير Data Deletion ⏳

**الملف**: `services/backend/internal/gdpr/data_deletion.go`

**الوظائف المطلوبة**:
```go
package gdpr

import (
    "context"
    "time"
)

type DeletionRequest struct {
    ID          int64     `json:"id"`
    TenantID    int64     `json:"tenant_id"`
    UserID      int64     `json:"user_id"`
    RequestedAt time.Time `json:"requested_at"`
    ProcessedAt *time.Time `json:"processed_at"`
    Status      string    `json:"status"` // pending, processing, completed, failed
    Reason      string    `json:"reason"`
}

type DataDeletionManager struct {
    db *sql.DB
}

func NewDataDeletionManager(db *sql.DB) *DataDeletionManager {
    return &DataDeletionManager{db: db}
}

// RequestDeletion creates a data deletion request
func (m *DataDeletionManager) RequestDeletion(ctx context.Context, tenantID, userID int64, reason string) error {
    // TODO: Implement
    // - Create deletion request
    // - Send confirmation email
    // - Schedule deletion job
    return nil
}

// ProcessDeletion processes a deletion request
func (m *DataDeletionManager) ProcessDeletion(ctx context.Context, requestID int64) error {
    // TODO: Implement
    // - Delete personal data
    // - Anonymize financial records
    // - Keep audit trail
    // - Update request status
    return nil
}

// DeletePersonalData deletes all personal data for a user
func (m *DataDeletionManager) DeletePersonalData(ctx context.Context, tenantID, userID int64) error {
    // TODO: Implement
    // Tables to delete from:
    // - users (personal info)
    // - user_consents
    // - notifications
    // - support_tickets (anonymize)
    
    // Tables to keep (anonymized):
    // - accounting_transactions (for audit)
    // - balance_history (for audit)
    // - subscriptions (anonymized)
    
    return nil
}

// AnonymizeFinancialRecords anonymizes financial records
func (m *DataDeletionManager) AnonymizeFinancialRecords(ctx context.Context, tenantID, userID int64) error {
    // TODO: Implement
    // - Replace user_id with anonymous ID
    // - Keep transaction amounts
    // - Keep timestamps
    // - Remove personal identifiers
    return nil
}
```

**Requirements**: 36.5, 36.6, 36.7

---

### 38.5 تطوير Data Export ⏳

**الملف**: `services/backend/internal/gdpr/data_export.go`

**الوظائف المطلوبة**:
```go
package gdpr

import (
    "context"
    "encoding/json"
)

type UserDataExport struct {
    PersonalInfo    map[string]interface{} `json:"personal_info"`
    Subscriptions   []interface{}          `json:"subscriptions"`
    Transactions    []interface{}          `json:"transactions"`
    Messages        []interface{}          `json:"messages"`
    Consents        []interface{}          `json:"consents"`
    ExportedAt      time.Time              `json:"exported_at"`
}

type DataExportManager struct {
    db *sql.DB
}

func NewDataExportManager(db *sql.DB) *DataExportManager {
    return &DataExportManager{db: db}
}

// ExportUserData exports all user data in JSON format
func (m *DataExportManager) ExportUserData(ctx context.Context, tenantID, userID int64) (*UserDataExport, error) {
    // TODO: Implement
    // - Collect all user data
    // - Format as JSON
    // - Include all tables
    // - Exclude sensitive system data
    return nil, nil
}

// GenerateExportFile generates a downloadable export file
func (m *DataExportManager) GenerateExportFile(ctx context.Context, tenantID, userID int64) (string, error) {
    // TODO: Implement
    // - Export data
    // - Create ZIP file
    // - Store temporarily
    // - Return download URL
    return "", nil
}
```

**API Handler**:
```go
// services/backend/internal/api/gdpr_handler.go
func (h *GDPRHandler) ExportData(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement
    // - Authenticate user
    // - Generate export
    // - Send download link via email
    // - Auto-delete after 7 days
}
```

**Requirements**: 36.8

---

### 38.6 تطبيق Fraud Prevention ⏳

**الملف**: `services/backend/internal/fraud/detector.go`

**الوظائف المطلوبة**:
```go
package fraud

import (
    "context"
    "time"
)

type FraudRule struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Enabled     bool   `json:"enabled"`
    Severity    string `json:"severity"` // low, medium, high, critical
}

type FraudAlert struct {
    ID          int64     `json:"id"`
    TenantID    int64     `json:"tenant_id"`
    UserID      int64     `json:"user_id"`
    RuleID      int64     `json:"rule_id"`
    Severity    string    `json:"severity"`
    Description string    `json:"description"`
    DetectedAt  time.Time `json:"detected_at"`
    Status      string    `json:"status"` // new, investigating, resolved, false_positive
}

type FraudDetector struct {
    db *sql.DB
}

func NewFraudDetector(db *sql.DB) *FraudDetector {
    return &FraudDetector{db: db}
}

// CheckTransaction checks a transaction for fraud
func (d *FraudDetector) CheckTransaction(ctx context.Context, transaction *Transaction) (bool, error) {
    // TODO: Implement fraud rules:
    // - Multiple failed payments
    // - Unusual transaction amounts
    // - Rapid subscription changes
    // - Geographic anomalies
    // - Velocity checks
    return false, nil
}

// LogSuspiciousActivity logs suspicious activity
func (d *FraudDetector) LogSuspiciousActivity(ctx context.Context, alert *FraudAlert) error {
    // TODO: Implement
    return nil
}
```

**Fraud Rules Examples**:
1. Multiple failed payment attempts (>3 in 1 hour)
2. Unusual transaction amounts (>10x average)
3. Rapid subscription changes (>5 in 1 day)
4. Geographic anomalies (login from different countries)
5. Velocity checks (too many API calls)

**Requirements**: 36.9

---

### 38.7 تطبيق Audit Logging ⏳

**ملاحظة**: Audit Logging موجود بالفعل في `services/backend/internal/audit/`

**التحسينات المطلوبة**:
```go
// services/backend/internal/audit/logger.go

// Add legal compliance events
const (
    EventConsentGranted    = "consent_granted"
    EventConsentRevoked    = "consent_revoked"
    EventDataExported      = "data_exported"
    EventDataDeleted       = "data_deleted"
    EventFraudDetected     = "fraud_detected"
    EventAccountSuspended  = "account_suspended"
)

// LogLegalEvent logs legal compliance events
func (l *Logger) LogLegalEvent(ctx context.Context, event *AuditEvent) error {
    // TODO: Implement
    // - Log with high priority
    // - Include all context
    // - Store for 7 years (legal requirement)
    // - Encrypt sensitive data
    return nil
}
```

**Retention Policy**:
- Legal events: 7 years
- Financial events: 7 years
- User activity: 1 year
- System logs: 90 days

**Requirements**: 36.10, 36.11

---

## خطوات التنفيذ الموصى بها

### المرحلة 1: الإعداد القانوني
1. استشارة مستشار قانوني
2. مراجعة القوانين المحلية
3. إعداد المحتوى القانوني
4. ترجمة معتمدة

### المرحلة 2: التطوير التقني
1. إنشاء صفحات شروط الخدمة والخصوصية
2. تطوير Consent Management
3. تطوير Data Deletion
4. تطوير Data Export
5. تطبيق Fraud Prevention
6. تحسين Audit Logging

### المرحلة 3: الاختبار
1. اختبار وظائف GDPR
2. اختبار Consent Management
3. اختبار Data Export/Deletion
4. اختبار Fraud Detection

### المرحلة 4: التوثيق
1. توثيق السياسات
2. توثيق الإجراءات
3. تدريب الفريق
4. إعداد دليل المستخدم

## الامتثال للمعايير

### GDPR (General Data Protection Regulation)
- ✅ Right to Access (Data Export)
- ✅ Right to Erasure (Data Deletion)
- ✅ Right to Rectification
- ✅ Consent Management
- ✅ Data Portability
- ✅ Privacy by Design

### CCPA (California Consumer Privacy Act)
- ✅ Right to Know
- ✅ Right to Delete
- ✅ Right to Opt-Out
- ✅ Non-Discrimination

### Local Laws (Yemen, Saudi Arabia)
- ⏳ Requires legal review
- ⏳ Requires localization

## الموارد المطلوبة

### قانونية
- مستشار قانوني متخصص
- مترجم قانوني معتمد
- مراجع قانوني

### تقنية
- مطور Backend (Go)
- مطور Frontend (React)
- مهندس أمن معلومات
- مهندس DevOps

### الوقت المقدر
- الإعداد القانوني: 2-3 أسابيع
- التطوير التقني: 1-2 أسابيع
- الاختبار: 1 أسبوع
- التوثيق: 1 أسبوع

**الإجمالي**: 5-7 أسابيع

## الخلاصة

المهام 38.1-38.7 تتطلب:
1. ✅ إطار عمل تقني (متوفر)
2. ⏳ محتوى قانوني (يتطلب مستشار)
3. ⏳ مراجعة قانونية (يتطلب متخصص)
4. ⏳ اختبار وتوثيق (يتطلب وقت)

**التوصية**: البدء بالاستشارة القانونية أولاً، ثم التطوير التقني بناءً على المتطلبات القانونية المحددة.

---

**Status**: 📋 Framework Ready - Requires Legal Content
**Date**: 2026-03-02
**Requirements**: 36.1-36.11
