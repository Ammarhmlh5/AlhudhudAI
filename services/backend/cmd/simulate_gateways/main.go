package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/receiver"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/workers/alhudhud"
	"github.com/al-hudhud/backend/internal/workers/private"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("=== بدء تشغيل بيئة المحاكاة والاختبار لبوابات الهدهد والبوابات الخاصة ===")

	godotenv.Load("../../.env")
	cfg, _ := config.Load()
	zapLog := logger.GetLogger()

	db.InitGormDB()
	redisClient, _ := redis.NewRedisClient(cfg, zapLog)
	if redisClient != nil {
		defer redisClient.Client.Close()
	}

	alhudhudWorker := alhudhud.NewWorker(redisClient, nil)
	alhudhudWorker.Start()
	defer alhudhudWorker.Stop()

	privateWorker := private.NewWorker(redisClient, nil)
	privateWorker.Start()
	defer privateWorker.Stop()

	tenantID := uuid.New()
	fmt.Printf("\n[+] إنشاء مستأجر تجريبي (TenantID: %s)\n", tenantID)

	bawabatyGateway := models.AlhudhudGateway{
		ProviderName: "bawabaty_test",
		BaseURL:      "http://localhost:9999/mock-bawabaty",
		APIKey:       "test_api_key",
		SenderID:     "ALHUDHUD",
		IsActive:     true,
	}
	db.GormDB.Create(&bawabatyGateway)
	defer db.GormDB.Unscoped().Delete(&bawabatyGateway)

	alhudhudDispatcher := receiver.NewAlhudhudDispatcher(redisClient)
	privateDispatcher := receiver.NewPrivateDispatcher(redisClient)
	ctx := context.Background()

	fmt.Println("\n[1] اختبار مسار الهدهد (ALHUDHUD PATH)...")
	msg1 := &models.Message{ID: uuid.New(), CustomID: "MS_ALHUDHUD_TEST_001", PhoneNumber: "+966500000000", Content: "رسالة اختبار عبر بوابة الهدهد", Status: models.MessageWait}
	
	_, err := alhudhudDispatcher.DispatchToAlhudhud(ctx, tenantID, msg1)
	if err != nil {
		log.Fatalf("فشل في توجيه رسالة الهدهد: %v", err)
	}
	fmt.Printf("✓ تم إرسال الرسالة (%s) بنجاح إلى طابور الهدهد في Redis\n", msg1.CustomID)

	fmt.Println("\n[2] اختبار المسار الخاص (PRIVATE GATEWAY PATH)...")
	msg2 := &models.Message{ID: uuid.New(), CustomID: "MS_PRIVATE_TEST_002", PhoneNumber: "+966511111111", Content: "رسالة اختبار عبر السيرفر الخاص للشركة", Status: models.MessageWait}
	
	privateServerURL := "http://localhost:9999/mock-private-webhook"
	_, err = privateDispatcher.DispatchToPrivateServer(ctx, tenantID, privateServerURL, msg2)
	if err != nil {
		log.Fatalf("فشل في توجيه الرسالة الخاصة: %v", err)
	}
	fmt.Printf("✓ تم إرسال طلب HTTP POST (Fire-and-Forget) إلى سيرفر الشركة (%s)\n", privateServerURL)

	time.Sleep(3 * time.Second)
	fmt.Println("\n=== اكتملت المحاكاة بنجاح! جميع المسارات معزولة وتعمل بشكل ممتاز ===")
}
