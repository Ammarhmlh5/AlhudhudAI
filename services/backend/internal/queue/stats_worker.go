package queue

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/riverqueue/river"
)

// StatisticsWorker aggregates system-wide metrics for the Admin Dashboard.
// It runs periodically to update the AdminStatistics table, ensuring O(1) reads.
type StatisticsWorker struct {
	river.WorkerDefaults[StatsArgs]
}

func (w *StatisticsWorker) Work(ctx context.Context, job *river.Job[StatsArgs]) error {
	var stats models.AdminStatistics
	stats.ID = 1 // Single row aggregation

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var count int64

	// 1. Tenant Metrics
	db.GormDB.Model(&models.Tenant{}).Count(&count)
	stats.TotalTenants = int(count)
	
	db.GormDB.Model(&models.Tenant{}).Where("is_active = ?", true).Count(&count)
	stats.ActiveTenants = int(count)

	// 2. Device Metrics
	db.GormDB.Model(&models.PairedDevice{}).Count(&count)
	stats.TotalDevices = int(count)
	
	db.GormDB.Model(&models.PairedDevice{}).Where("status = ?", "ONLINE").Count(&count)
	stats.ActiveDevices = int(count)

	// 3. Balance Metrics
	db.GormDB.Model(&models.Tenant{}).Select("SUM(balance)").Scan(&stats.TotalBalance)

	// 4. Message Metrics
	db.GormDB.Model(&models.Message{}).Where("status = ?", models.MessagePending).Count(&count)
	stats.PendingMessages = int(count)
	
	db.GormDB.Model(&models.Message{}).Where("status = ?", models.MessageQueued).Count(&count)
	stats.QueuedMessages = int(count)
	
	db.GormDB.Model(&models.Message{}).Where("status = ?", models.MessageProcessing).Count(&count)
	stats.ProcessingMessages = int(count)

	// Today's Stats
	db.GormDB.Model(&models.Message{}).Where("created_at >= ?", todayStart).Count(&count)
	stats.TotalMessagesToday = int(count)
	
	db.GormDB.Model(&models.Message{}).Where("created_at >= ? AND status = ?", todayStart, models.MessageSent).Count(&count)
	stats.SentMessagesToday = int(count)
	
	db.GormDB.Model(&models.Message{}).Where("created_at >= ? AND status = ?", todayStart, models.MessageFailed).Count(&count)
	stats.FailedMessagesToday = int(count)

	// 5. Raw Event Metrics
	db.GormDB.Model(&models.GlobalIncomingEvent{}).Where("created_at >= ?", todayStart).Count(&count)
	stats.TotalEventsToday = int(count)
	
	db.GormDB.Model(&models.GlobalIncomingEvent{}).Where("created_at >= ? AND status = ?", todayStart, "COMPLETED").Count(&count)
	stats.ParsedEventsToday = int(count)
	
	db.GormDB.Model(&models.GlobalIncomingEvent{}).Where("created_at >= ? AND status = ?", todayStart, "FAILED").Count(&count)
	stats.FailedEventsToday = int(count)

	stats.LastUpdated = now

	// 6. UPSERT into admin_statistics
	return db.GormDB.Save(&stats).Error
}

func (StatisticsWorker) Kind() string { return "stats_aggregation" }
