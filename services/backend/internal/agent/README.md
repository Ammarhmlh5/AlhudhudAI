# Agent System - نظام الوكلاء

نظام إدارة الوكلاء والموظفين لمنصة الهدهد.

## الوصف

يوفر هذا النظام إمكانية لشركات البرمجة (الوكلاء) من إدارة عملائهم (المستأجرين) من خلال واجهة برمجية محددة الصلاحيات.

## الميزات

- ✅ تسجيل دخول الوكلاء والموظفين
- ✅ إنشاء وإدارة الموظفين (للوكيل فقط)
- ✅ إنشاء المستأجرين (للوكيل والموظفين)
- ✅ سداد الفواتير (للوكيل والموظفين)
- ❌ منع عمليات الحذف والتعطيل (تتم من المنصة الرئيسية)

## الملفات

```
internal/agent/
├── auth.go       # المصادقة و JWT
├── handler.go    # نقاط النهاية
├── router.go     # إعداد المسارات
├── scope.go      # العزل والصلاحيات
├── service.go    # منطق الأعمال
├── setup.go      # التهيئة
├── integration.go # أدوات التكامل
└── README.md     # هذا الملف
```

## الاستخدام

```go
// في main.go
import (
    "github.com/al-hudhud/backend/internal/agent"
)

func main() {
    // ...
    agent.Initialize(router, db)
    // ...
}
```

## المسارات

### المصادقة
- `POST /api/v1/agents/auth/login` - تسجيل دخول الوكيل
- `POST /api/v1/agents/auth/employees/login` - تسجيل دخول الموظف
- `POST /api/v1/agents/auth/change-password` - تغيير كلمة المرور

### الموظفون
- `POST /api/v1/agents/employees` - إنشاء موظف (للوكيل فقط)
- `GET /api/v1/agents/employees` - قائمة الموظفين
- `DELETE /api/v1/agents/employees/:id` - حذف موظف

### المستأجرون
- `POST /api/v1/agents/tenants` - إنشاء مستأجر
- `GET /api/v1/agents/tenants` - قائمة المستأجرين

### المدفوعات
- `POST /api/v1/agents/payments` - سداد فاتورة
- `GET /api/v1/agents/payments` - قائمة المدفوعات

## المصادقة

النظام يستخدم JWT tokens مع الصلاحيات التالية:
- `user_id`: معرف المستخدم
- `agent_id`: معرف الوكيل
- `user_type`: نوع المستخدم (agent/employee)
- `role`: الدور (admin/staff)

## العزل

الموظف يرى فقط مستأجرين الوكيل الذي ينتمي إليه.

## الامان

- ✅ تشفير كلمات المرور بـ bcrypt
- ✅ JWT tokens مع تواريخ انتهاء
- ✅ Rate limiting للحماية
- ✅ تسجيل الأنشطة

## الاختبار

```bash
go test ./internal/agent/...
```