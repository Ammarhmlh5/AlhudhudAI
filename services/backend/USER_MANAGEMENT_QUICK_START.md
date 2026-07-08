# User Management للبوابات المخصصة - دليل البدء السريع

## نظرة عامة

نظام إدارة المستخدمين للبوابات المخصصة يسمح للشركات المؤسسية بإضافة مستخدمين متعددين لبواباتهم مع أدوار وصلاحيات مختلفة.

## المزايا الرئيسية

✅ **إضافة مستخدمين متعددين** لكل بوابة مخصصة  
✅ **ثلاثة أدوار**: Admin، Operator، Viewer  
✅ **صلاحيات دقيقة** قابلة للتخصيص  
✅ **RBAC** (Role-Based Access Control)  
✅ **Middleware** للتحقق من الصلاحيات  
✅ **عزل تام** بين الشركات (Multi-Tenancy)

---

## الأدوار والصلاحيات

### 1. Admin (المدير)
**الصلاحيات الكاملة:**
- ✅ إرسال الرسائل (`send_messages`)
- ✅ عرض التقارير (`view_reports`)
- ✅ إدارة المستخدمين (`manage_users`)
- ✅ إدارة البوابة (`manage_gateway`)
- ✅ إدارة API Keys (`manage_api_keys`)
- ✅ عرض الإحصائيات (`view_statistics`)
- ✅ تصدير التقارير (`export_reports`)

### 2. Operator (المشغّل)
**صلاحيات محدودة:**
- ✅ إرسال الرسائل (`send_messages`)
- ✅ عرض التقارير (`view_reports`)
- ✅ عرض الإحصائيات (`view_statistics`)
- ✅ تصدير التقارير (`export_reports`)
- ❌ إدارة المستخدمين
- ❌ إدارة البوابة
- ❌ إدارة API Keys

### 3. Viewer (المشاهد)
**صلاحيات القراءة فقط:**
- ✅ عرض التقارير (`view_reports`)
- ✅ عرض الإحصائيات (`view_statistics`)
- ❌ إرسال الرسائل
- ❌ تصدير التقارير
- ❌ إدارة المستخدمين
- ❌ إدارة البوابة

---

## الاستخدام

### 1. إنشاء User Manager

```go
import (
    "github.com/yourusername/sms-gateway/internal/gateway"
    "gorm.io/gorm"
)

// إنشاء User Manager
userManager := gateway.NewUserManager(db)
```

### 2. إضافة مستخدم

```go
import (
    "context"
    "github.com/google/uuid"
)

// إضافة مستخدم بدور Admin
req := &gateway.AddUserRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    UserID:    userID,
    Role:      gateway.RoleAdmin,
}

gatewayUser, err := userManager.AddUser(context.Background(), req)
if err != nil {
    // معالجة الخطأ
}
```

### 3. إضافة مستخدم بصلاحيات مخصصة

```go
// إضافة مستخدم بصلاحيات مخصصة
customPermissions := map[string]interface{}{
    gateway.PermissionSendMessages:  true,
    gateway.PermissionViewReports:   true,
    gateway.PermissionManageUsers:   false,
    gateway.PermissionManageGateway: false,
}

req := &gateway.AddUserRequest{
    TenantID:    tenantID,
    GatewayID:   gatewayID,
    UserID:      userID,
    Role:        gateway.RoleOperator,
    Permissions: customPermissions,
}

gatewayUser, err := userManager.AddUser(context.Background(), req)
```

### 4. تحديث الصلاحيات

```go
// تحديث صلاحيات مستخدم موجود
newPermissions := map[string]interface{}{
    gateway.PermissionSendMessages:  false,
    gateway.PermissionViewReports:   true,
    gateway.PermissionManageUsers:   true,
}

req := &gateway.UpdatePermissionsRequest{
    TenantID:    tenantID,
    GatewayID:   gatewayID,
    UserID:      userID,
    Permissions: newPermissions,
}

gatewayUser, err := userManager.UpdatePermissions(context.Background(), req)
```

### 5. التحقق من الصلاحيات

```go
// التحقق من صلاحية معينة
req := &gateway.CheckGatewayPermissionRequest{
    TenantID:   tenantID,
    GatewayID:  gatewayID,
    UserID:     userID,
    Permission: gateway.PermissionSendMessages,
}

hasPermission, err := userManager.CheckGatewayPermission(context.Background(), req)
if err != nil {
    // معالجة الخطأ
}

if hasPermission {
    // المستخدم لديه الصلاحية
} else {
    // المستخدم ليس لديه الصلاحية
}
```

### 6. تفعيل/تعطيل مستخدم

```go
// تعطيل مستخدم
req := &gateway.UpdateUserStatusRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    UserID:    userID,
    IsActive:  false,
}

err := userManager.UpdateUserStatus(context.Background(), req)

// إعادة تفعيل مستخدم
req.IsActive = true
err = userManager.UpdateUserStatus(context.Background(), req)
```

### 7. عرض جميع المستخدمين

```go
// عرض جميع مستخدمي البوابة
req := &gateway.GetUsersByGatewayRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
}

users, err := userManager.GetUsersByGateway(context.Background(), req)
if err != nil {
    // معالجة الخطأ
}

for _, user := range users {
    fmt.Printf("User: %s, Role: %s, Active: %v\n", 
        user.UserID, user.Role, user.IsActive)
}
```

### 8. حذف مستخدم

```go
// حذف مستخدم من البوابة
req := &gateway.RemoveUserRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    UserID:    userID,
}

err := userManager.RemoveUser(context.Background(), req)
```

---

## استخدام Middleware

### 1. إنشاء Middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/yourusername/sms-gateway/internal/middleware"
)

// إنشاء middleware
permissionMiddleware := middleware.NewGatewayPermissionMiddleware(userManager)
```

### 2. التحقق من صلاحية معينة

```go
// حماية endpoint بصلاحية معينة
router.POST("/gateway/:gateway_id/messages", 
    permissionMiddleware.RequirePermission(gateway.PermissionSendMessages),
    sendMessageHandler,
)
```

### 3. التحقق من دور معين

```go
// السماح فقط للـ Admin
router.POST("/gateway/:gateway_id/users", 
    permissionMiddleware.RequireAdminRole(),
    addUserHandler,
)

// السماح للـ Admin أو Operator
router.GET("/gateway/:gateway_id/reports", 
    permissionMiddleware.RequireOperatorOrAdmin(),
    getReportsHandler,
)

// السماح لأي دور
router.GET("/gateway/:gateway_id/statistics", 
    permissionMiddleware.RequireAnyRole(),
    getStatisticsHandler,
)
```

### 4. التحقق من أدوار متعددة

```go
// السماح لأدوار محددة
router.PUT("/gateway/:gateway_id/settings", 
    permissionMiddleware.RequireRole("admin", "operator"),
    updateSettingsHandler,
)
```

---

## أمثلة API Handlers

### مثال 1: إضافة مستخدم

```go
func AddUserHandler(c *gin.Context) {
    var req gateway.AddUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // استخراج tenant_id من context (تم تعيينه بواسطة middleware)
    tenantID := c.GetString("tenant_id")
    req.TenantID = uuid.MustParse(tenantID)

    // استخراج gateway_id من URL
    gatewayID, _ := strconv.Atoi(c.Param("gateway_id"))
    req.GatewayID = gatewayID

    gatewayUser, err := userManager.AddUser(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    gatewayUser,
    })
}
```

### مثال 2: عرض المستخدمين

```go
func GetGatewayUsersHandler(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    gatewayID, _ := strconv.Atoi(c.Param("gateway_id"))

    req := &gateway.GetUsersByGatewayRequest{
        TenantID:  uuid.MustParse(tenantID),
        GatewayID: gatewayID,
    }

    users, err := userManager.GetUsersByGateway(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    users,
        "count":   len(users),
    })
}
```

### مثال 3: تحديث الصلاحيات

```go
func UpdateUserPermissionsHandler(c *gin.Context) {
    var req gateway.UpdatePermissionsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    tenantID := c.GetString("tenant_id")
    req.TenantID = uuid.MustParse(tenantID)

    gatewayID, _ := strconv.Atoi(c.Param("gateway_id"))
    req.GatewayID = gatewayID

    userID := c.Param("user_id")
    req.UserID = uuid.MustParse(userID)

    gatewayUser, err := userManager.UpdatePermissions(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    gatewayUser,
    })
}
```

---

## Routes المقترحة

```go
// إنشاء router group للبوابات
gatewayRoutes := router.Group("/api/v1/gateway/:gateway_id")
{
    // إدارة المستخدمين (Admin فقط)
    userRoutes := gatewayRoutes.Group("/users")
    userRoutes.Use(permissionMiddleware.RequireAdminRole())
    {
        userRoutes.POST("", AddUserHandler)
        userRoutes.GET("", GetGatewayUsersHandler)
        userRoutes.PUT("/:user_id/permissions", UpdateUserPermissionsHandler)
        userRoutes.PUT("/:user_id/status", UpdateUserStatusHandler)
        userRoutes.DELETE("/:user_id", RemoveUserHandler)
    }

    // إرسال الرسائل (Admin أو Operator)
    gatewayRoutes.POST("/messages", 
        permissionMiddleware.RequirePermission(gateway.PermissionSendMessages),
        SendMessageHandler,
    )

    // عرض التقارير (جميع الأدوار)
    gatewayRoutes.GET("/reports", 
        permissionMiddleware.RequireAnyRole(),
        GetReportsHandler,
    )

    // تصدير التقارير (Admin أو Operator)
    gatewayRoutes.GET("/reports/export", 
        permissionMiddleware.RequirePermission(gateway.PermissionExportReports),
        ExportReportsHandler,
    )

    // إدارة البوابة (Admin فقط)
    gatewayRoutes.PUT("/settings", 
        permissionMiddleware.RequirePermission(gateway.PermissionManageGateway),
        UpdateGatewaySettingsHandler,
    )
}
```

---

## العزل بين الشركات (Multi-Tenancy)

النظام يطبق عزل تام بين الشركات:

1. **Row Level Security (RLS)** في قاعدة البيانات
2. **Tenant Validation** في جميع العمليات
3. **Indexes مركبة** تبدأ بـ `tenant_id`
4. **Middleware** للتحقق من `tenant_id`

```sql
-- مثال على RLS Policy
CREATE POLICY tenant_isolation_policy ON gateway_users
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

---

## الاختبارات

### تشغيل الاختبارات

```bash
# تشغيل جميع اختبارات User Manager
go test -v ./internal/gateway -run TestAddUser
go test -v ./internal/gateway -run TestUpdatePermissions
go test -v ./internal/gateway -run TestCheckGatewayPermission
go test -v ./internal/gateway -run TestRoleBasedAccessControl

# تشغيل اختبارات Middleware
go test -v ./internal/middleware -run TestRequirePermission
go test -v ./internal/middleware -run TestRequireRole
```

---

## الملفات المرتبطة

- **User Manager**: `internal/gateway/user_manager.go`
- **User Manager Tests**: `internal/gateway/user_manager_test.go`
- **Middleware**: `internal/middleware/gateway_permission.go`
- **Middleware Tests**: `internal/middleware/gateway_permission_test.go`
- **Migration**: `migrations/023_gateway_users_table.sql`
- **README**: `migrations/README_GATEWAY_USERS.md`

---

## المتطلبات المحققة

✅ **Requirement 11.1**: دعم إضافة مستخدمين متعددين  
✅ **Requirement 11.2**: دعم أدوار مختلفة (admin، operator، viewer)  
✅ **Requirement 11.3**: تحديد صلاحيات حسب الدور  
✅ **Requirement 11.4**: تخصيص صلاحيات دقيقة  
✅ **Requirement 11.5**: تفعيل/تعطيل المستخدمين  
✅ **Requirement 11.6**: تسجيل تاريخ الإضافة  
✅ **Requirement 11.7**: صلاحيات Admin الكاملة  
✅ **Requirement 11.8**: صلاحيات Operator المحدودة  
✅ **Requirement 11.9**: صلاحيات Viewer للقراءة فقط

---

## الخطوات التالية

1. ✅ **المهمة 20.1**: تطوير وظيفة إضافة مستخدم - **مكتمل**
2. ✅ **المهمة 20.2**: تطوير وظيفة تخصيص الصلاحيات - **مكتمل**
3. ✅ **المهمة 20.3**: تطوير Middleware للتحقق من الصلاحيات - **مكتمل**
4. ⏳ **المهمة 21**: تطوير API Key Management - **قادم**

---

## الدعم والمساعدة

للمزيد من المعلومات:
- راجع `migrations/README_GATEWAY_USERS.md` للتفاصيل الكاملة
- راجع `.kiro/specs/tariff-pricing-system/requirements.md` للمتطلبات
- راجع `.kiro/specs/tariff-pricing-system/design.md` للتصميم

