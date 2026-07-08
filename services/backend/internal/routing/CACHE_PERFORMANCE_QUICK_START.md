# Cache Performance Tests - Quick Start

## 🚀 تشغيل سريع

### Windows (PowerShell)
```powershell
cd services/backend
.\test-cache-performance.ps1
```

### Linux/Mac (Bash)
```bash
cd services/backend
chmod +x test-cache-performance.sh
./test-cache-performance.sh
```

## 📋 المتطلبات

1. ✅ PostgreSQL يعمل على `localhost:5432`
2. ✅ Redis يعمل على `localhost:6379`
3. ✅ Migrations تم تشغيلها

## 🎯 الاختبارات المتاحة

| الاختبار | الأمر | الهدف |
|---------|-------|--------|
| **الكل** | `--test all` | جميع الاختبارات |
| **Hit Rate** | `--test hitrate` | Cache Hit Rate > 95% ⭐ |
| **DB Load** | `--test dbload` | تقليل Queries بنسبة 99% ⭐ |
| **Latency** | `--test latency` | مقارنة الأداء |
| **Concurrent** | `--test concurrent` | الضغط المتزامن |
| **TTL** | `--test ttl` | انتهاء الصلاحية |
| **Multi-Tenant** | `--test multitenant` | العزل |

⭐ = اختبارات المتطلب 5.7 الأساسية

## 💡 أمثلة

### تشغيل جميع الاختبارات
```bash
./test-cache-performance.sh
```

### تشغيل اختبار محدد
```bash
# Cache Hit Rate (الأهم)
./test-cache-performance.sh --test hitrate

# Database Load Reduction (المتطلب 5.7)
./test-cache-performance.sh --test dbload
```

### تشغيل مع تفاصيل
```bash
./test-cache-performance.sh --test all --verbose
```

## ✅ معايير النجاح

| المقياس | القيمة المطلوبة |
|---------|-----------------|
| Cache Hit Rate | > 95% |
| Database Query Reduction | > 99% |
| Cache Hit Latency | < 2ms |
| Performance Improvement | > 5x |

## 📊 فهم النتائج

### نتيجة ناجحة ✅
```
Cache Performance Results:
  Cache Hits: 95
  Hit Rate: 95.00%
  Query Reduction: 99.00%
  Average Latency: 1.5ms
```

### نتيجة فاشلة ❌
```
Cache hit rate should be greater than 95%
Expected: > 95.0
Actual: 85.0
```

## 🔧 استكشاف الأخطاء

### PostgreSQL لا يعمل
```bash
docker-compose up -d postgres
```

### Redis لا يعمل
```bash
docker-compose up -d redis
```

### Migrations لم تُشغل
```bash
cd services/backend
go run cmd/migrate/main.go up
```

## 📚 المزيد من المعلومات

- **دليل كامل:** `CACHE_PERFORMANCE_TEST_GUIDE.md`
- **ملخص التنفيذ:** `.kiro/specs/message-lifecycle-rebuild/TASK_14.7_EXECUTION_SUMMARY.md`
- **الكود:** `cache_performance_test.go`

---

**المتطلب:** 5.7 - تقليل Database Queries بنسبة 99%
