# دليل البدء السريع - اختبارات الأداء

## الخطوات السريعة

### 1. تثبيت الأدوات المطلوبة

#### Windows
```powershell
# تثبيت k6
choco install k6

# تثبيت PostgreSQL client (إذا لم يكن مثبتاً)
choco install postgresql
```

#### Linux
```bash
# تثبيت k6
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# تثبيت PostgreSQL client
sudo apt-get install postgresql-client

# تثبيت Apache Bench
sudo apt-get install apache2-utils
```

### 2. إعداد بيانات الاختبار

```bash
cd performance-tests/seed-data

# Linux/Mac
chmod +x run-seed.sh
./run-seed.sh

# Windows
run-seed.bat
```

هذا سينشئ:
- ✅ 10 شركات اختبار برصيد 100,000 لكل منها
- ✅ 50 قناة إرسال (5 لكل شركة)

### 3. تشغيل الاختبارات

#### اختبار واحد

```bash
# اختبار API Gateway (Response Time < 10ms)
k6 run k6-tests/api-gateway-test.js

# اختبار معالجة الدفعات
k6 run k6-tests/batch-processing-test.js

# اختبار معدل الإرسال (5000+ رسالة/دقيقة)
k6 run k6-tests/worker-throughput-test.js

# اختبار الضغط الشديد (10,000 رسالة/دقيقة)
k6 run k6-tests/stress-test.js
```

#### جميع الاختبارات

```bash
cd performance-tests

# Linux/Mac
chmod +x run-all-tests.sh
./run-all-tests.sh

# Windows
# استخدم Git Bash أو WSL
bash run-all-tests.sh
```

### 4. قراءة النتائج

النتائج تُحفظ في مجلد `results/`:
- `api-gateway-test-summary.json`
- `batch-processing-test-summary.json`
- `worker-throughput-test-summary.json`
- `stress-test-summary.json`

## معايير النجاح

### ✅ API Gateway
- Response Time < 10ms (P95)
- Throughput >= 1000 req/sec
- Error Rate < 0.1%

### ✅ Batch Processing
- 100 رسالة في دفعة أسرع 50x من معالجة فردية
- Latency < 200ms

### ✅ Worker Throughput
- معدل إرسال >= 5000 رسالة/دقيقة
- Success Rate >= 99%
- Average Latency < 2 ثانية

### ✅ Stress Test
- النظام يتحمل 10,000 رسالة/دقيقة
- Error Rate < 5%
- لا أخطاء حرجة

## استكشاف الأخطاء

### مشكلة: k6 غير موجود
```bash
# تحقق من التثبيت
k6 version

# إذا لم يكن مثبتاً، راجع خطوة التثبيت أعلاه
```

### مشكلة: فشل الاتصال بقاعدة البيانات
```bash
# تحقق من تشغيل PostgreSQL
docker ps | grep postgres

# تحقق من المتغيرات البيئية
echo $DB_HOST
echo $DB_PORT
echo $DB_NAME
```

### مشكلة: فشل الاتصال بـ API
```bash
# تحقق من تشغيل Backend
curl http://localhost:8080/health

# تحقق من المتغير البيئي
echo $API_URL
```

### مشكلة: Response Time أعلى من المتوقع
- تحقق من حمل قاعدة البيانات
- تحقق من Redis performance
- تحقق من عدد Workers
- راقب استهلاك CPU و Memory

## نصائح مهمة

1. **لا تشغل الاختبارات على بيئة الإنتاج**
2. **استخدم بيئة Staging منفصلة**
3. **راقب استهلاك الموارد أثناء الاختبار**
4. **احفظ نتائج كل اختبار للمقارنة**
5. **شغّل الاختبارات في أوقات مختلفة لقياس الاستقرار**

## المتغيرات البيئية

```bash
# تخصيص إعدادات الاختبار
export API_URL=http://localhost:8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=sms_gateway
export DB_USER=postgres
export DB_PASSWORD=postgres
```

## الدعم

إذا واجهت أي مشاكل:
1. راجع ملف `README.md` الكامل
2. تحقق من logs النظام
3. راجع نتائج الاختبارات في مجلد `results/`
