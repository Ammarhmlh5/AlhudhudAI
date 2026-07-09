# دليل النسخ الاحتياطي والاسترجاع - المهمة 37

## نظرة عامة

دليل شامل لإدارة النسخ الاحتياطي واسترجاع البيانات.

## المهمة 37.1: Backup Strategy

### سكريبت النسخ الاحتياطي اليومي

```bash
#!/bin/bash
# scripts/backup-tariff-data.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/tariff-system"
DB_NAME="sms_gateway"

# نسخ احتياطي لجداول التعرفات
pg_dump -h localhost -U postgres -t tariffs -t packages -t subscriptions \
    -t rollover_messages -t dedicated_gateways -t resource_quotas \
    -t resource_usage -t gateway_users -t gateway_api_keys \
    -t gateway_statistics -t accounting_transactions -t balance_history \
    -t billing_invoices -t promo_codes -t referrals -t payment_transactions \
    $DB_NAME > "$BACKUP_DIR/tariff_backup_$DATE.sql"

# ضغط النسخة
gzip "$BACKUP_DIR/tariff_backup_$DATE.sql"

# رفع إلى S3
aws s3 cp "$BACKUP_DIR/tariff_backup_$DATE.sql.gz" \
    s3://my-backups/tariff-system/daily/

echo "Backup completed: tariff_backup_$DATE.sql.gz"
```

### Cron Job للنسخ التلقائي

```bash
# إضافة إلى crontab
# نسخ احتياطي يومي الساعة 2 صباحاً
0 2 * * * /path/to/scripts/backup-tariff-data.sh

# نسخ احتياطي أسبوعي الأحد الساعة 3 صباحاً
0 3 * * 0 /path/to/scripts/backup-tariff-data-weekly.sh

# نسخ احتياطي شهري أول يوم من الشهر
0 4 1 * * /path/to/scripts/backup-tariff-data-monthly.sh
```

## المهمة 37.2: Retention Policy

### سكريبت تنظيف النسخ القديمة

```bash
#!/bin/bash
# scripts/cleanup-old-backups.sh

BACKUP_DIR="/backups/tariff-system"

# حذف النسخ اليومية الأقدم من 30 يوم
find "$BACKUP_DIR/daily" -name "*.sql.gz" -mtime +30 -delete

# حذف النسخ الأسبوعية الأقدم من 90 يوم
find "$BACKUP_DIR/weekly" -name "*.sql.gz" -mtime +90 -delete

# حذف النسخ الشهرية الأقدم من 365 يوم
find "$BACKUP_DIR/monthly" -name "*.sql.gz" -mtime +365 -delete

echo "Old backups cleaned up"
```

## المهمة 37.3: Backup Encryption

### تشفير النسخ الاحتياطية

```bash
#!/bin/bash
# scripts/encrypt-backup.sh

BACKUP_FILE=$1
ENCRYPTION_KEY="/secure/keys/backup.key"

# تشفير باستخدام GPG
gpg --symmetric --cipher-algo AES256 \
    --passphrase-file "$ENCRYPTION_KEY" \
    "$BACKUP_FILE"

# حذف النسخة غير المشفرة
rm "$BACKUP_FILE"

echo "Backup encrypted: $BACKUP_FILE.gpg"
```

## المهمة 37.4: Backup Storage

### رفع إلى S3 مع Versioning

```bash
# تفعيل Versioning على S3 Bucket
aws s3api put-bucket-versioning \
    --bucket my-backups \
    --versioning-configuration Status=Enabled

# رفع مع Metadata
aws s3 cp backup.sql.gz.gpg \
    s3://my-backups/tariff-system/daily/ \
    --metadata "backup-date=$(date +%Y-%m-%d),type=daily"
```

## المهمة 37.5: Backup Validation

### اختبار صحة النسخة

```bash
#!/bin/bash
# scripts/validate-backup.sh

BACKUP_FILE=$1
TEST_DB="test_restore_db"

# إنشاء قاعدة بيانات اختبار
createdb $TEST_DB

# استرجاع النسخة
gunzip -c "$BACKUP_FILE" | psql $TEST_DB

# التحقق من الجداول
TABLES=$(psql -t $TEST_DB -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public'")

if [ "$TABLES" -gt 0 ]; then
    echo "Backup validation successful: $TABLES tables restored"
else
    echo "Backup validation failed: No tables found"
    exit 1
fi

# حذف قاعدة البيانات الاختبارية
dropdb $TEST_DB
```

## المهمة 37.6: Recovery Procedures

### سكريبت الاسترجاع

```bash
#!/bin/bash
# scripts/restore-tariff-data.sh

BACKUP_FILE=$1
DB_NAME="sms_gateway"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# تأكيد من المستخدم
read -p "This will restore $DB_NAME from $BACKUP_FILE. Continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled"
    exit 0
fi

# إيقاف التطبيق
systemctl stop tariff-system

# نسخ احتياطي للحالة الحالية
pg_dump $DB_NAME > "/backups/pre-restore-$(date +%Y%m%d_%H%M%S).sql"

# فك الضغط والتشفير
gunzip -c "$BACKUP_FILE" | gpg --decrypt | psql $DB_NAME

# إعادة تشغيل التطبيق
systemctl start tariff-system

echo "Restore completed from $BACKUP_FILE"
```

## المهمة 37.7: Disaster Recovery Plan

### خطة الاسترجاع من الكوارث

```markdown
# docs/DISASTER_RECOVERY_PLAN.md

# خطة الاسترجاع من الكوارث

## السيناريوهات

### 1. فقدان قاعدة البيانات الكاملة

**RTO:** 4 ساعات
**RPO:** 24 ساعة

**خطوات الاسترجاع:**
1. إنشاء قاعدة بيانات جديدة
2. استرجاع آخر نسخة احتياطية يومية
3. تطبيق Transaction Logs إن وجدت
4. التحقق من سلامة البيانات
5. إعادة تشغيل التطبيق

### 2. فشل السيرفر الكامل

**RTO:** 2 ساعة
**RPO:** 1 ساعة

**خطوات الاسترجاع:**
1. تفعيل Failover إلى السيرفر الاحتياطي
2. تحديث DNS
3. التحقق من عمل التطبيق
4. إصلاح السيرفر الأساسي

### 3. فساد البيانات

**RTO:** 6 ساعات
**RPO:** 24 ساعة

**خطوات الاسترجاع:**
1. تحديد نطاق الفساد
2. استرجاع الجداول المتأثرة فقط
3. التحقق من سلامة البيانات
4. إعادة تشغيل التطبيق
```

## Checklist للتحقق

- [ ] النسخ الاحتياطي اليومي يعمل
- [ ] النسخ مشفرة ومخزنة بأمان
- [ ] Retention Policy مطبّقة
- [ ] اختبار الاسترجاع ناجح
- [ ] خطة DR موثقة ومختبرة

**Requirements Covered:** 25.1-25.11, 28.8, 28.9
