# Admin Interface Enhancements - نظام إدارة شامل

## نظرة عامة

نظام إدارة متكامل لمنصة Hudhud SMS يوفر:
- تقييم تلقائي للشركات بناءً على مؤشرات الأداء
- مراقبة صحة السيرفرات المخصصة
- توزيع ذكي للرسائل على أجهزة متعددة
- محرك توجيه Signal/Push مع مراقبة التوقيت
- واجهة إدارة متقدمة
- نظام مراقبة وتنبيهات شامل

## الحالة الحالية

✅ **المشروع مكتمل وجاهز للنشر!**

- **نسبة الإنجاز**: 100% (المهام الأساسية)
- **المراحل المكتملة**: 9/9
- **الحالة**: جاهز للنشر في الإنتاج

## التقنيات المستخدمة

### Backend
- **Language**: Go (Golang) 1.21+
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **API**: RESTful APIs
- **Authentication**: JWT

### Frontend
- **Framework**: React 18+
- **Language**: TypeScript
- **UI Library**: Tailwind CSS + shadcn/ui
- **State Management**: Zustand
- **Data Fetching**: React Query
- **Charts**: Recharts
- **i18n**: i18next (Arabic + English)

### Monitoring
- **Metrics**: Prometheus
- **Dashboards**: Grafana
- **Alerting**: Alertmanager

### Deployment
- **Containerization**: Docker
- **Orchestration**: Docker Compose
- **CI/CD**: Ready for integration

## البنية المعمارية

```
┌─────────────────────────────────────────────────────────────┐
│                     Admin Web Interface                      │
│                  (React + TypeScript)                        │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTP/WebSocket
┌────────────────────────▼────────────────────────────────────┐
│                    Admin Backend API                         │
│                      (Go/Golang)                             │
├──────────────────────────────────────────────────────────────┤
│  • Rating System      • Health Monitor   • Distribution      │
│  • Routing Engine     • Signal/Push      • Alert System      │
└────┬──────────────┬──────────────┬────────────────┬─────────┘
     │              │              │                │
┌────▼────┐   ┌────▼────┐   ┌────▼────┐     ┌────▼────┐
│PostgreSQL│   │  Redis  │   │Prometheus│     │ Grafana │
└─────────┘   └─────────┘   └──────────┘     └─────────┘
```

## الميزات الرئيسية

### 1. نظام تقييم الشركات
- حساب تلقائي للتقييم بناءً على:
  - معدل النجاح (Success Rate)
  - حجم الإرسال (Sending Volume)
  - الالتزام بالدفع (Payment Compliance)
  - جودة الخدمة (Service Quality)
- تصنيف الشركات: Excellent, Good, Average, Poor
- تاريخ التقييمات
- تنبيهات عند تغيير التقييم

### 2. مراقبة صحة السيرفرات
- فحص تلقائي كل 30 ثانية
- تصنيف الحالة: Healthy, Degraded, Unhealthy
- قياس Response Time و Error Rate
- تنبيهات فورية عند تغيير الحالة
- تاريخ الفحوصات

### 3. نظام توزيع الرسائل
- استراتيجيات متعددة:
  - Round Robin
  - Load Balancing
  - Priority-Based
  - Custom Rules
- إدارة الأجهزة
- معالجة الفشل التلقائية
- مقاييس التوزيع

### 4. محرك Signal/Push
- مراقبة التوقيت بين الرسائل
- كشف الشذوذ (Anomaly Detection)
- تطبيق تلقائي للإجراءات
- إرسال تكيفي حسب حمل السيرفر
- معالجة تقارير التسليم بالدفعات

### 5. نظام التنبيهات
- مستويات متعددة: Critical, Warning, Info
- إزالة التكرار
- قنوات متعددة: Email, SMS, In-app
- إدارة التنبيهات: Acknowledge, Resolve

### 6. واجهة الإدارة
- Dashboard شامل
- إدارة الشركات
- مراقبة السيرفرات
- إدارة التنبيهات
- إعدادات التوزيع
- مراقبة Signal/Push
- دعم RTL للعربية
- Responsive Design

## البدء السريع

### المتطلبات
- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM على الأقل
- 20GB مساحة تخزين

### التثبيت

1. **Clone المشروع**
```bash
git clone <repository-url>
cd hudhud-sms
```

2. **إعداد Environment Variables**
```bash
cp .env.example .env
# قم بتعديل .env وإضافة القيم المطلوبة
```

3. **Build و Start الخدمات**
```bash
docker-compose build
docker-compose up -d
```

4. **تشغيل Migrations**
```bash
docker-compose exec admin-backend /app/admin-backend migrate up
```

5. **التحقق من التثبيت**
```bash
# Admin Backend
curl http://localhost:8081/health

# Admin Web
curl http://localhost:3001/health

# Metrics
curl http://localhost:9091/metrics
```

### الوصول للخدمات

- **Admin Web**: http://localhost:3001
- **Admin API**: http://localhost:8081
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

## التوثيق

### للمستخدمين
- [دليل المستخدم](../../docs/ADMIN_USER_GUIDE.md) - كيفية استخدام واجهة الإدارة

### للمطورين
- [دليل المطورين](../../docs/ADMIN_DEVELOPER_GUIDE.md) - البنية المعمارية والتطوير
- [API Reference](../../docs/ADMIN_API_REFERENCE.md) - توثيق جميع الـ APIs

### للنشر
- [دليل النشر](../../docs/ADMIN_DEPLOYMENT_GUIDE.md) - خطوات النشر التفصيلية
- [Docker Guide](../../docker/README.md) - استخدام Docker
- [Monitoring Guide](../../prometheus/README.md) - إعداد المراقبة

### التقارير
- [Project Completion Summary](./PROJECT_COMPLETION_SUMMARY.md) - ملخص إكمال المشروع
- [Final Project Status](./FINAL_PROJECT_STATUS.md) - الحالة النهائية
- [Phase 9 Complete](./PHASE_9_DEPLOYMENT_COMPLETE.md) - تقرير المرحلة 9
- [Phase 8 Complete](./PHASE_8_TESTING_COMPLETE.md) - تقرير المرحلة 8
- [Deployment Checklist](./DEPLOYMENT_CHECKLIST.md) - قائمة فحص النشر

## هيكل المشروع

```
.
├── services/
│   └── backend/
│       ├── cmd/admin/              # Admin backend entry point
│       ├── internal/
│       │   ├── admin/
│       │   │   ├── rating/         # Rating system
│       │   │   ├── health/         # Health monitoring
│       │   │   ├── routing/        # Routing engine
│       │   │   ├── distribution/   # Distribution system
│       │   │   ├── signalpush/     # Signal/Push system
│       │   │   ├── alerts/         # Alert system
│       │   │   └── metrics/        # Metrics collection
│       │   ├── api/                # API handlers
│       │   └── monitoring/         # Prometheus metrics
│       ├── migrations/             # Database migrations
│       │   └── rollback/           # Rollback scripts
│       └── scripts/                # Utility scripts
│
├── apps/
│   └── admin-web/
│       ├── src/
│       │   ├── pages/              # React pages
│       │   ├── components/         # React components
│       │   ├── hooks/              # Custom hooks
│       │   └── api/                # API client
│       └── Dockerfile              # Frontend Docker image
│
├── docs/                           # Documentation
│   ├── ADMIN_API_REFERENCE.md
│   ├── ADMIN_USER_GUIDE.md
│   ├── ADMIN_DEVELOPER_GUIDE.md
│   └── ADMIN_DEPLOYMENT_GUIDE.md
│
├── grafana/
│   └── dashboards/                 # Grafana dashboards
│
├── prometheus/
│   ├── admin_alert_rules.yml      # Alert rules
│   └── prometheus.yml              # Prometheus config
│
└── docker-compose.yml              # Docker Compose config
```

## API Endpoints

### Dashboard
- `GET /api/admin/dashboard/overview` - نظرة عامة

### Company Rating
- `GET /api/admin/companies/{tenant_id}/rating` - تقييم شركة
- `GET /api/admin/companies/ratings` - قائمة التقييمات
- `GET /api/admin/companies/{tenant_id}/rating/history` - تاريخ التقييم

### Server Health
- `GET /api/admin/servers` - قائمة السيرفرات
- `GET /api/admin/servers/{server_id}` - تفاصيل سيرفر
- `GET /api/admin/servers/{server_id}/health/history` - تاريخ الصحة

### Routing
- `GET /api/admin/routing/{tenant_id}` - إعدادات التوجيه
- `PUT /api/admin/routing/{tenant_id}` - تحديث التوجيه
- `GET /api/admin/routing/{tenant_id}/metrics` - مقاييس التوجيه

### Distribution
- `GET /api/admin/distribution/{tenant_id}` - استراتيجية التوزيع
- `PUT /api/admin/distribution/{tenant_id}` - تحديث الاستراتيجية
- `GET /api/admin/distribution/{tenant_id}/devices` - قائمة الأجهزة
- `GET /api/admin/distribution/{tenant_id}/metrics` - مقاييس التوزيع

### Alerts
- `GET /api/admin/alerts` - قائمة التنبيهات
- `GET /api/admin/alerts/{alert_id}` - تفاصيل تنبيه
- `PUT /api/admin/alerts/{alert_id}/acknowledge` - الإقرار بتنبيه
- `PUT /api/admin/alerts/{alert_id}/resolve` - حل تنبيه

### Signal/Push
- `GET /api/admin/signalpush/{tenant_id}/config` - إعدادات التوقيت
- `PUT /api/admin/signalpush/{tenant_id}/config` - تحديث الإعدادات
- `GET /api/admin/signalpush/{tenant_id}/statistics` - الإحصائيات
- `GET /api/admin/signalpush/{tenant_id}/violations` - الانتهاكات
- `POST /api/admin/signalpush/{tenant_id}/force-mode` - فرض وضع

## المراقبة

### Prometheus Metrics
- 40+ metric لمراقبة جميع مكونات النظام
- API performance metrics
- Server health metrics
- Distribution metrics
- Timing violation metrics
- Alert metrics

### Grafana Dashboards
- Admin System Dashboard (12 panels)
- Real-time monitoring
- Auto-refresh كل 30 ثانية

### Alert Rules
- 29 alert rule في 8 groups
- Critical و Warning levels
- Integration مع Alertmanager

## الاختبارات

### Integration Tests
```bash
cd services/backend
go test ./internal/integration/... -v
```

### Performance Tests
```bash
cd performance-tests
k6 run admin-load-test.js
k6 run admin-stress-test.js
```

### Security Tests
```bash
cd services/backend
go test ./internal/security/... -v
```

## المساهمة

### Development Setup
1. Fork المشروع
2. إنشاء branch جديد
3. تطوير الميزة
4. كتابة الاختبارات
5. إرسال Pull Request

### Code Style
- Backend: Go formatting standards
- Frontend: ESLint + Prettier
- Commits: Conventional Commits

## الترخيص

[License Type] - راجع ملف LICENSE للتفاصيل

## الدعم

- **Documentation**: راجع مجلد `docs/`
- **Issues**: افتح issue في GitHub
- **Email**: support@example.com

## الفريق

- **Backend Development**: [Team]
- **Frontend Development**: [Team]
- **DevOps**: [Team]
- **QA**: [Team]

## الإصدارات

### v1.0.0 (Current)
- ✅ نظام تقييم الشركات
- ✅ مراقبة صحة السيرفرات
- ✅ نظام توزيع الرسائل
- ✅ محرك Signal/Push
- ✅ نظام التنبيهات
- ✅ واجهة الإدارة
- ✅ نظام المراقبة

### Roadmap
- [ ] Property-Based Tests (اختياري)
- [ ] Additional Unit Tests (اختياري)
- [ ] E2E Tests للـ Frontend
- [ ] Advanced Analytics
- [ ] Machine Learning Integration

## الشكر والتقدير

شكراً لجميع المساهمين في هذا المشروع!

---

**الحالة**: ✅ مكتمل وجاهز للنشر
**الإصدار**: v1.0.0
**آخر تحديث**: $(date +%Y-%m-%d)
