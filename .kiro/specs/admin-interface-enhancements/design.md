# Admin Interface Enhancements - Design Document

## Overview

This design document specifies the technical architecture for comprehensive admin interface enhancements that enable advanced company monitoring, rating, dedicated server routing, multi-device message distribution, and integration with the new Signal/Push messaging routing system.

### Design Scope

This design covers:
- High-level system architecture
- Database schema design for admin features
- API endpoints for admin functions
- Signal/Push messaging routing engine integration
- Company rating and performance monitoring systems
- Dedicated server routing mechanisms
- Multi-device message distribution strategies
- Admin dashboard UI architecture
- Integration with existing systems (tariff-pricing-system, dedicated gateways, resource management)

### Design Objectives

1. **Comprehensive Monitoring**: Real-time visibility into company performance and system health
2. **Intelligent Routing**: Automatic routing to dedicated servers with health monitoring
3. **Optimized Distribution**: Smart message distribution across multiple devices
4. **Signal/Push Integration**: Seamless integration with timing-aware message routing
5. **Scalability**: Support for 1000+ companies with real-time metrics
6. **Security**: Role-based access control and audit trails


## Architecture

### System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Admin Dashboard (React)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐│
│  │  Companies   │  │   Servers    │  │ Distribution │  │   Alerts    ││
│  │   Monitor    │  │   Monitor    │  │   Config     │  │   Center    ││
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬──────┘│
└─────────┼──────────────────┼──────────────────┼──────────────────┼───────┘
          │                  │                  │                  │
          │ HTTP/REST        │ HTTP/REST        │ HTTP/REST        │ WebSocket
          │                  │                  │                  │
┌─────────┴──────────────────┴──────────────────┴──────────────────┴───────┐
│                      Backend API Gateway (Go)                             │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                    Admin API Endpoints                              │  │
│  │  • Company Rating APIs    • Server Health APIs                     │  │
│  │  • Distribution APIs      • Alert Management APIs                  │  │
│  └────────────────────┬───────────────────────────────────────────────┘  │
│                       │                                                    │
│  ┌────────────────────┴───────────────────────────────────────────────┐  │
│  │                   Admin Business Logic Layer                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │  │
│  │  │   Company    │  │   Server     │  │ Distribution │            │  │
│  │  │   Rating     │  │   Health     │  │   Strategy   │            │  │
│  │  │   Engine     │  │   Monitor    │  │   Manager    │            │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │  │
│  └────────────────────┬───────────────────────────────────────────────┘  │
│                       │                                                    │
│  ┌────────────────────┴───────────────────────────────────────────────┐  │
│  │              Signal/Push Routing Integration Layer                  │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │  │
│  │  │   Message    │  │   Timing     │  │   Anomaly    │            │  │
│  │  │   Router     │  │   Monitor    │  │   Detector   │            │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │  │
│  │  │   Adaptive   │  │   Delivery   │  │   Priority   │            │  │
│  │  │   Sender     │  │   Report     │  │   Queue      │            │  │
│  │  │              │  │   Processor  │  │   Manager    │            │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │  │
│  └────────────────────┬───────────────────────────────────────────────┘  │
│                       │                                                    │
│  ┌────────────────────┴───────────────────────────────────────────────┐  │
│  │                   Core Integration Layer                            │  │
│  │  • Tariff System Integration    • Gateway Management               │  │
│  │  • Resource Management           • Balance Management               │  │
│  │  • Statistics Aggregation        • Notification Service             │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────┘
          │                  │                  │                  │
          │                  │                  │                  │
┌─────────┴──────────────────┴──────────────────┴──────────────────┴───────┐
│                      Data & Cache Layer                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │
│  │  PostgreSQL  │  │    Redis     │  │  Prometheus  │  │   Grafana   │ │
│  │  (Primary)   │  │   (Cache)    │  │  (Metrics)   │  │ (Dashboard) │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘ │
└───────────────────────────────────────────────────────────────────────────┘
```


### Component Architecture

#### 1. Company Rating Engine

**Purpose**: Calculate and maintain company ratings based on performance metrics

**Components**:
- **Metrics Collector**: Gathers performance data from multiple sources
- **Rating Calculator**: Computes ratings using weighted formulas
- **Category Classifier**: Assigns companies to rating categories
- **History Manager**: Maintains 12-month rating history

**Data Flow**:
```
Performance Metrics → Metrics Collector → Rating Calculator → Category Classifier → Database
                                                                      ↓
                                                              Alert System (on category change)
```

#### 2. Server Health Monitor

**Purpose**: Monitor dedicated server health and availability

**Components**:
- **Health Checker**: Periodic health check execution (every 30 seconds)
- **Status Classifier**: Categorizes server status (healthy/degraded/unhealthy)
- **Metrics Tracker**: Tracks response time and error rates
- **Alert Trigger**: Generates alerts on status changes

**Health Check Criteria**:
- Response time < 200ms → Healthy
- Response time 200-500ms OR error rate 1-5% → Degraded
- Response time > 500ms OR error rate > 5% → Unhealthy

#### 3. Message Distribution Strategy Manager

**Purpose**: Manage message distribution across multiple devices

**Supported Strategies**:
1. **Round Robin**: Sequential device selection with fair distribution
2. **Load Balancing**: Distribution based on current device load
3. **Priority-based**: Distribution based on device priority levels
4. **Custom Rules**: Administrator-defined distribution rules

**Components**:
- **Strategy Selector**: Chooses appropriate strategy per company
- **Device Pool Manager**: Maintains available device pools
- **Load Tracker**: Monitors device load in real-time
- **Failure Handler**: Excludes failed devices temporarily (5 minutes)


#### 4. Signal/Push Routing Engine

**Purpose**: Route messages through Signal or Push paths based on timing compliance

**Components**:
- **Message Queue Manager**: PostgreSQL-based persistent queue
- **Timing Monitor**: Tracks inter-message timing compliance
- **Statistical Analyzer**: Builds statistical profiles per company
- **Anomaly Detector**: Detects abnormal sending patterns
- **Auto-Enforcement System**: Automatically switches to Push mode on violations
- **Adaptive Sender**: Adjusts sending rate based on server load
- **Delivery Report Processor**: Handles delivery reports in batches

**Routing Decision Flow**:
```
Message Received
    ↓
[Timing Monitor] → Check inter-message gap
    ↓
Gap sufficient? 
    ├─→ Yes → Signal Path (direct send if device online)
    └─→ No → [Record Violation]
              ↓
         [Statistical Analyzer] → Calculate metrics
              ↓
         [Anomaly Detector] → Determine anomaly level
              ↓
         Severe Anomaly?
              ├─→ Yes → [Auto-Enforcement] → Force Push Mode
              └─→ No → Continue with current mode
```

**Timing Violation Handling**:
- **Minor Anomaly** (Z-Score > 1.5): Log only, continue monitoring
- **Moderate Anomaly** (Z-Score > 2): Send warning to company
- **Severe Anomaly** (Z-Score > 3 OR Violation Rate > 50%): Force Push mode

**Push Mode Processing**:
1. Batch messages in PostgreSQL queue
2. Monitor server load (CPU, Memory)
3. Adjust sending rate adaptively:
   - High load (CPU > 80%): Reduce rate by 50%
   - Low load (CPU < 50%): Increase rate by 100%
   - Normal load: Maintain current rate
4. Send in batches with rate limiting
5. Process delivery reports in batches (100 reports per transaction)


#### 5. Dedicated Server Routing

**Purpose**: Route company messages to their dedicated servers

**Components**:
- **Routing Configuration Manager**: Manages routing rules per company
- **Server Registry**: Maintains dedicated server endpoints
- **Health-Aware Router**: Routes only to healthy servers
- **Fallback Handler**: Handles routing failures

**Routing Logic**:
```go
func RouteMessage(companyID uuid.UUID, message *Message) error {
    // 1. Check if company has dedicated server
    config := GetRoutingConfig(companyID)
    if config == nil || !config.HasDedicatedServer {
        return RouteToSharedInfrastructure(message)
    }
    
    // 2. Check server health
    server := GetDedicatedServer(config.ServerID)
    if server.Status != "healthy" {
        LogError("Dedicated server unhealthy", server.ID)
        return ErrServerUnavailable
    }
    
    // 3. Route to dedicated server
    return SendToDedicatedServer(server, message)
}
```

**No Fallback Policy**: If dedicated server is unavailable, return error to caller (no automatic fallback to shared infrastructure to maintain isolation)


## Components and Interfaces

### 1. Company Rating System

#### Rating Calculation Formula

```go
type RatingWeights struct {
    SuccessRate      float64 // 40%
    SendingVolume    float64 // 20%
    PaymentCompliance float64 // 25%
    ServiceQuality   float64 // 15%
}

func CalculateRating(metrics *PerformanceMetrics, weights *RatingWeights) float64 {
    rating := 0.0
    
    // Success Rate (0-100)
    successRate := (metrics.SuccessfulMessages / metrics.TotalMessages) * 100
    rating += successRate * weights.SuccessRate
    
    // Sending Volume (normalized to 0-100)
    volumeScore := NormalizeVolume(metrics.TotalMessages)
    rating += volumeScore * weights.SendingVolume
    
    // Payment Compliance (0-100)
    paymentScore := (metrics.OnTimePayments / metrics.TotalPayments) * 100
    rating += paymentScore * weights.PaymentCompliance
    
    // Service Quality (0-100)
    qualityScore := CalculateQualityScore(metrics)
    rating += qualityScore * weights.ServiceQuality
    
    return rating
}
```

#### Rating Categories

```go
type RatingCategory string

const (
    Excellent RatingCategory = "excellent" // >= 90%
    Good      RatingCategory = "good"      // 75-89%
    Average   RatingCategory = "average"   // 60-74%
    Poor      RatingCategory = "poor"      // < 60%
)

func ClassifyRating(rating float64) RatingCategory {
    switch {
    case rating >= 90:
        return Excellent
    case rating >= 75:
        return Good
    case rating >= 60:
        return Average
    default:
        return Poor
    }
}
```

#### Rating Update Trigger

```go
type RatingUpdateTrigger struct {
    EventType string // "message_sent", "message_failed", "payment_received"
    TenantID  uuid.UUID
    Timestamp time.Time
}

func (r *RatingEngine) OnEvent(trigger *RatingUpdateTrigger) {
    // Debounce: Update rating max once per 5 seconds per company
    if r.ShouldUpdate(trigger.TenantID) {
        go r.UpdateRating(trigger.TenantID)
    }
}
```


### 2. Message Distribution System

#### Distribution Strategies

**Round Robin Strategy**:
```go
type RoundRobinStrategy struct {
    devices      []*Device
    currentIndex int
    mu           sync.Mutex
}

func (rr *RoundRobinStrategy) SelectDevice() (*Device, error) {
    rr.mu.Lock()
    defer rr.mu.Unlock()
    
    if len(rr.devices) == 0 {
        return nil, ErrNoDevicesAvailable
    }
    
    device := rr.devices[rr.currentIndex]
    rr.currentIndex = (rr.currentIndex + 1) % len(rr.devices)
    
    return device, nil
}
```

**Load Balancing Strategy**:
```go
type LoadBalancingStrategy struct {
    devices []*Device
}

func (lb *LoadBalancingStrategy) SelectDevice() (*Device, error) {
    if len(lb.devices) == 0 {
        return nil, ErrNoDevicesAvailable
    }
    
    // Sort by current load (ascending)
    sort.Slice(lb.devices, func(i, j int) bool {
        return lb.devices[i].CurrentLoad < lb.devices[j].CurrentLoad
    })
    
    return lb.devices[0], nil
}
```

**Priority-based Strategy**:
```go
type PriorityStrategy struct {
    devices []*Device
}

func (ps *PriorityStrategy) SelectDevice() (*Device, error) {
    if len(ps.devices) == 0 {
        return nil, ErrNoDevicesAvailable
    }
    
    // Sort by priority (descending), then by load (ascending)
    sort.Slice(ps.devices, func(i, j int) bool {
        if ps.devices[i].Priority != ps.devices[j].Priority {
            return ps.devices[i].Priority > ps.devices[j].Priority
        }
        return ps.devices[i].CurrentLoad < ps.devices[j].CurrentLoad
    })
    
    return ps.devices[0], nil
}
```

#### Device Failure Handling

```go
type DeviceFailureHandler struct {
    excludedDevices map[uuid.UUID]time.Time
    excludeDuration time.Duration // 5 minutes
    mu              sync.RWMutex
}

func (dfh *DeviceFailureHandler) OnDeviceFailure(deviceID uuid.UUID) {
    dfh.mu.Lock()
    defer dfh.mu.Unlock()
    
    dfh.excludedDevices[deviceID] = time.Now().Add(dfh.excludeDuration)
}

func (dfh *DeviceFailureHandler) IsDeviceExcluded(deviceID uuid.UUID) bool {
    dfh.mu.RLock()
    defer dfh.mu.RUnlock()
    
    excludeUntil, exists := dfh.excludedDevices[deviceID]
    if !exists {
        return false
    }
    
    if time.Now().After(excludeUntil) {
        delete(dfh.excludedDevices, deviceID)
        return false
    }
    
    return true
}
```


### 3. Signal/Push Timing Monitor

#### Timing Violation Detection

```go
type TimingMonitor struct {
    redis    *redis.Client
    db       *gorm.DB
    analyzer *StatisticalAnalyzer
}

func (tm *TimingMonitor) CheckMessage(ctx context.Context, msg *Message) error {
    // 1. Get last message timestamp
    lastTimeKey := fmt.Sprintf("timing:tenant:%s:last_message", msg.TenantID)
    lastTimeStr, err := tm.redis.Get(ctx, lastTimeKey).Result()
    
    if err == redis.Nil {
        // First message, no violation
        tm.redis.Set(ctx, lastTimeKey, msg.Timestamp.Unix(), time.Hour)
        return nil
    }
    
    lastTime := time.Unix(lastTimeStr, 0)
    gap := msg.Timestamp.Sub(lastTime)
    
    // 2. Get expected gap for this tenant
    config := tm.GetTimingConfig(msg.TenantID)
    expectedGap := config.GetExpectedGap(msg.Type) // Signal: 2s, Push: 0.5s
    
    // 3. Check violation
    if gap < expectedGap {
        violation := &TimingViolation{
            TenantID:    msg.TenantID,
            MessageID:   msg.ID,
            MessageType: msg.Type,
            ExpectedGap: expectedGap,
            ActualGap:   gap,
            Timestamp:   time.Now(),
        }
        
        // Record violation
        tm.RecordViolation(ctx, violation)
        
        // Update statistics and check for anomaly
        stats := tm.analyzer.UpdateStatistics(ctx, msg.TenantID, msg.Type)
        anomaly := tm.analyzer.DetectAnomaly(ctx, stats)
        
        // Take action if severe
        if anomaly.Level >= SevereAnomaly {
            return tm.EnforceAction(ctx, msg.TenantID, anomaly)
        }
    }
    
    // 4. Update last message time
    tm.redis.Set(ctx, lastTimeKey, msg.Timestamp.Unix(), time.Hour)
    
    return nil
}
```

#### Statistical Analysis

```go
type StatisticalAnalyzer struct {
    db *gorm.DB
}

func (sa *StatisticalAnalyzer) UpdateStatistics(ctx context.Context, tenantID uuid.UUID, msgType string) *TenantStatistics {
    // Calculate current MPM (Messages Per Minute)
    currentMPM := sa.CalculateCurrentMPM(ctx, tenantID, time.Minute)
    
    // Get historical baselines
    dailyAvg := sa.GetAverageMPM(ctx, tenantID, 24*time.Hour)
    dailyPeak := sa.GetPeakMPM(ctx, tenantID, 24*time.Hour)
    weeklyAvg := sa.GetAverageMPM(ctx, tenantID, 7*24*time.Hour)
    weeklyPeak := sa.GetPeakMPM(ctx, tenantID, 7*24*time.Hour)
    
    // Calculate standard deviation
    stdDev := sa.CalculateStdDev(ctx, tenantID, time.Hour)
    
    stats := &TenantStatistics{
        TenantID:         tenantID,
        MessageType:      msgType,
        CurrentMPM:       currentMPM,
        CurrentStdDev:    stdDev,
        DailyAverageMPM:  dailyAvg,
        DailyPeakMPM:     dailyPeak,
        WeeklyAverageMPM: weeklyAvg,
        WeeklyPeakMPM:    weeklyPeak,
        LastUpdated:      time.Now(),
    }
    
    // Save to database
    sa.db.Save(stats)
    
    return stats
}
```


#### Anomaly Detection Algorithm

```go
type AnomalyDetector struct{}

type AnomalyLevel int

const (
    NoAnomaly       AnomalyLevel = 0
    MinorAnomaly    AnomalyLevel = 1
    ModerateAnomaly AnomalyLevel = 2
    SevereAnomaly   AnomalyLevel = 3
)

func (ad *AnomalyDetector) DetectAnomaly(stats *TenantStatistics) *AnomalyResult {
    result := &AnomalyResult{
        TenantID: stats.TenantID,
        Level:    NoAnomaly,
    }
    
    // 1. Calculate Z-Score
    zScore := 0.0
    if stats.CurrentStdDev > 0 {
        zScore = (stats.CurrentMPM - stats.DailyAverageMPM) / stats.CurrentStdDev
    }
    result.ZScore = zScore
    
    // 2. Calculate Peak Ratio
    peakRatio := 0.0
    if stats.DailyPeakMPM > 0 {
        peakRatio = stats.CurrentMPM / stats.DailyPeakMPM
    }
    result.PeakRatio = peakRatio
    
    // 3. Get violation rate
    violationRate := ad.GetViolationRate(stats.TenantID, time.Hour)
    result.ViolationRate = violationRate
    
    // 4. Determine anomaly level
    if zScore > 3 || peakRatio > 2 || violationRate > 50 {
        result.Level = SevereAnomaly
        result.Reason = "Severe anomaly detected: possible abuse"
    } else if zScore > 2 || peakRatio > 1.5 || violationRate > 30 {
        result.Level = ModerateAnomaly
        result.Reason = "Moderate anomaly detected: suspicious behavior"
    } else if zScore > 1.5 || violationRate > 10 {
        result.Level = MinorAnomaly
        result.Reason = "Minor anomaly detected: monitoring required"
    }
    
    return result
}
```

#### Auto-Enforcement System

```go
type AutoEnforcement struct {
    db            *gorm.DB
    notifier      *NotificationService
}

func (ae *AutoEnforcement) EnforceAction(ctx context.Context, tenantID uuid.UUID, anomaly *AnomalyResult) error {
    switch anomaly.Level {
    case MinorAnomaly:
        // Log only
        ae.LogAnomaly(ctx, tenantID, anomaly)
        
    case ModerateAnomaly:
        // Send warning
        ae.SendWarning(ctx, tenantID, anomaly)
        ae.LogAnomaly(ctx, tenantID, anomaly)
        
    case SevereAnomaly:
        // Force Push mode
        action := &EnforcementAction{
            TenantID:      tenantID,
            ActionType:    "force_push",
            Reason:        anomaly.Reason,
            AnomalyLevel:  int(anomaly.Level),
            ViolationRate: anomaly.ViolationRate,
            ZScore:        anomaly.ZScore,
            PeakRatio:     anomaly.PeakRatio,
            PreviousMode:  "signal",
            NewMode:       "push",
            CreatedAt:     time.Now(),
        }
        
        // Update routing config
        err := ae.ForcePushMode(ctx, tenantID)
        if err != nil {
            return err
        }
        
        // Record action
        ae.db.Create(action)
        
        // Notify company and admins
        ae.notifier.NotifyCompany(tenantID, "Your account has been switched to Push mode due to timing violations")
        ae.notifier.NotifyAdmins(fmt.Sprintf("Company %s forced to Push mode", tenantID))
        
        ae.LogAnomaly(ctx, tenantID, anomaly)
    }
    
    return nil
}
```


### 4. Adaptive Sending System

#### Server Load Monitoring

```go
type ServerLoadMonitor struct {
    cpuThreshold    float64 // 80%
    memoryThreshold float64 // 85%
}

type ServerLoad struct {
    CPU    float64 // percentage
    Memory float64 // percentage
}

func (slm *ServerLoadMonitor) GetCurrentLoad() *ServerLoad {
    // Get system metrics
    cpuPercent := GetCPUUsage()
    memoryPercent := GetMemoryUsage()
    
    return &ServerLoad{
        CPU:    cpuPercent,
        Memory: memoryPercent,
    }
}

func (slm *ServerLoadMonitor) IsOverloaded(load *ServerLoad) bool {
    return load.CPU > slm.cpuThreshold || load.Memory > slm.memoryThreshold
}
```

#### Adaptive Rate Adjustment

```go
type AdaptiveSender struct {
    minBatchSize    int // 10
    maxBatchSize    int // 1000
    currentBatchSize int
    
    minSendRate     int // 100 msg/sec
    maxSendRate     int // 10000 msg/sec
    currentSendRate int
    
    loadMonitor     *ServerLoadMonitor
}

func (as *AdaptiveSender) AdjustSendingRate(load *ServerLoad) {
    switch {
    case load.CPU > 80 || load.Memory > 85:
        // High load - reduce rate
        as.currentBatchSize = max(as.minBatchSize, as.currentBatchSize/2)
        as.currentSendRate = max(as.minSendRate, as.currentSendRate/2)
        
    case load.CPU < 50 && load.Memory < 60:
        // Low load - increase rate
        as.currentBatchSize = min(as.maxBatchSize, as.currentBatchSize*2)
        as.currentSendRate = min(as.maxSendRate, as.currentSendRate*2)
        
    default:
        // Normal load - maintain current rate
    }
}

func (as *AdaptiveSender) SendBatch(ctx context.Context, messages []*Message) error {
    // Check server load
    load := as.loadMonitor.GetCurrentLoad()
    as.AdjustSendingRate(load)
    
    // Split into sub-batches
    subBatches := as.SplitIntoBatches(messages, as.currentBatchSize)
    
    for _, subBatch := range subBatches {
        // Check if overloaded before each batch
        if as.loadMonitor.IsOverloaded(load) {
            time.Sleep(time.Second) // Backoff
            load = as.loadMonitor.GetCurrentLoad()
        }
        
        // Send sub-batch
        err := as.SendSubBatch(ctx, subBatch)
        if err != nil {
            return err
        }
        
        // Rate limit delay
        delay := time.Second / time.Duration(as.currentSendRate/as.currentBatchSize)
        time.Sleep(delay)
    }
    
    return nil
}
```


### 5. Delivery Report Batch Processor

```go
type DeliveryReportProcessor struct {
    reportBuffer  chan *DeliveryReport
    bufferSize    int           // 10000
    batchSize     int           // 100
    flushInterval time.Duration // 5 seconds
    workerCount   int           // 10
    db            *gorm.DB
}

func (drp *DeliveryReportProcessor) Start(ctx context.Context) {
    // Start workers
    for i := 0; i < drp.workerCount; i++ {
        go drp.worker(ctx)
    }
    
    // Start periodic flush
    go drp.periodicFlush(ctx)
}

func (drp *DeliveryReportProcessor) worker(ctx context.Context) {
    reports := make([]*DeliveryReport, 0, drp.batchSize)
    
    for {
        select {
        case report := <-drp.reportBuffer:
            reports = append(reports, report)
            
            if len(reports) >= drp.batchSize {
                drp.ProcessBatch(ctx, reports)
                reports = reports[:0]
            }
            
        case <-ctx.Done():
            if len(reports) > 0 {
                drp.ProcessBatch(ctx, reports)
            }
            return
        }
    }
}

func (drp *DeliveryReportProcessor) ProcessBatch(ctx context.Context, reports []*DeliveryReport) error {
    return drp.db.Transaction(func(tx *gorm.DB) error {
        for _, report := range reports {
            // 1. Move from waiting_messages to delivered_messages
            err := drp.MoveToDelivered(tx, report)
            if err != nil {
                return err
            }
            
            // 2. Update message status
            err = drp.UpdateMessageStatus(tx, report)
            if err != nil {
                return err
            }
            
            // 3. Handle refund if failed
            if report.Status == "failed" {
                err = drp.RefundBalance(tx, report)
                if err != nil {
                    return err
                }
            }
        }
        
        return nil
    })
}
```


## Data Models

### Database Schema

#### 1. Company Ratings Table

```sql
CREATE TABLE company_ratings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Rating components
    success_rate DECIMAL(5,2) NOT NULL,
    sending_volume BIGINT NOT NULL,
    payment_compliance DECIMAL(5,2) NOT NULL,
    service_quality_score DECIMAL(5,2) NOT NULL,
    
    -- Calculated rating
    overall_rating DECIMAL(5,2) NOT NULL,
    rating_category VARCHAR(20) NOT NULL, -- 'excellent', 'good', 'average', 'poor'
    
    -- Metadata
    calculation_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMP,
    
    -- Indexes
    INDEX idx_company_ratings_tenant (tenant_id, calculation_timestamp DESC),
    INDEX idx_company_ratings_category (rating_category),
    INDEX idx_company_ratings_valid (valid_from, valid_to),
    
    -- Constraints
    CONSTRAINT check_rating_range CHECK (overall_rating >= 0 AND overall_rating <= 100),
    CONSTRAINT check_category CHECK (rating_category IN ('excellent', 'good', 'average', 'poor'))
);

-- Enable Row Level Security
ALTER TABLE company_ratings ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON company_ratings
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 2. Performance Metrics Table

```sql
CREATE TABLE performance_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Time period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- 'hourly', 'daily', 'monthly'
    
    -- Message metrics
    total_messages BIGINT NOT NULL DEFAULT 0,
    successful_messages BIGINT NOT NULL DEFAULT 0,
    failed_messages BIGINT NOT NULL DEFAULT 0,
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_messages > 0 
        THEN (successful_messages::DECIMAL / total_messages) * 100 
        ELSE 0 END
    ) STORED,
    
    -- Delivery metrics
    average_delivery_time_ms INTEGER,
    p95_delivery_time_ms INTEGER,
    p99_delivery_time_ms INTEGER,
    
    -- Error metrics
    error_count INTEGER NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_messages > 0 
        THEN (error_count::DECIMAL / total_messages) * 100 
        ELSE 0 END
    ) STORED,
    
    -- Payment metrics
    total_payments INTEGER NOT NULL DEFAULT 0,
    on_time_payments INTEGER NOT NULL DEFAULT 0,
    payment_compliance DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_payments > 0 
        THEN (on_time_payments::DECIMAL / total_payments) * 100 
        ELSE 0 END
    ) STORED,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_performance_metrics_tenant_period (tenant_id, period_start DESC),
    INDEX idx_performance_metrics_type (period_type),
    UNIQUE INDEX idx_performance_metrics_unique (tenant_id, period_start, period_type)
);

ALTER TABLE performance_metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON performance_metrics
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```


#### 3. Routing Configurations Table

```sql
CREATE TABLE routing_configurations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    
    -- Routing type
    routing_type VARCHAR(20) NOT NULL DEFAULT 'shared', -- 'shared', 'dedicated'
    
    -- Dedicated server config
    dedicated_server_id UUID REFERENCES dedicated_servers(id),
    dedicated_server_url VARCHAR(500),
    dedicated_server_api_key VARCHAR(255),
    
    -- Routing metrics
    total_requests BIGINT NOT NULL DEFAULT 0,
    successful_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    average_latency_ms INTEGER,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_health_check TIMESTAMP,
    health_status VARCHAR(20), -- 'healthy', 'degraded', 'unhealthy'
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_routing_config_tenant (tenant_id),
    INDEX idx_routing_config_type (routing_type),
    INDEX idx_routing_config_server (dedicated_server_id)
);

ALTER TABLE routing_configurations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON routing_configurations
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 4. Distribution Strategies Table

```sql
CREATE TABLE distribution_strategies (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    
    -- Strategy configuration
    strategy_type VARCHAR(50) NOT NULL, -- 'round_robin', 'load_balancing', 'priority', 'custom'
    strategy_config JSONB, -- Strategy-specific configuration
    
    -- Custom rules (for custom strategy)
    custom_rules JSONB,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_distribution_strategies_tenant (tenant_id),
    INDEX idx_distribution_strategies_type (strategy_type),
    
    CONSTRAINT check_strategy_type CHECK (strategy_type IN ('round_robin', 'load_balancing', 'priority', 'custom'))
);

ALTER TABLE distribution_strategies ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON distribution_strategies
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 5. Phone Devices Table

```sql
CREATE TABLE phone_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Device information
    device_name VARCHAR(255) NOT NULL,
    device_model VARCHAR(255),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    
    -- Device configuration
    priority_level INTEGER NOT NULL DEFAULT 1, -- 1-5 (5 = highest)
    max_concurrent_messages INTEGER NOT NULL DEFAULT 10,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'offline', -- 'online', 'offline', 'failed', 'excluded'
    last_seen TIMESTAMP,
    last_failure TIMESTAMP,
    excluded_until TIMESTAMP,
    
    -- Metrics
    total_messages_sent BIGINT NOT NULL DEFAULT 0,
    successful_messages BIGINT NOT NULL DEFAULT 0,
    failed_messages BIGINT NOT NULL DEFAULT 0,
    current_load INTEGER NOT NULL DEFAULT 0,
    average_processing_time_ms INTEGER,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_phone_devices_tenant (tenant_id),
    INDEX idx_phone_devices_status (status),
    INDEX idx_phone_devices_priority (priority_level DESC),
    INDEX idx_phone_devices_load (current_load ASC)
);

ALTER TABLE phone_devices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON phone_devices
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```


#### 6. Distribution Metrics Table

```sql
CREATE TABLE distribution_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    device_id UUID NOT NULL REFERENCES phone_devices(id),
    
    -- Time period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    
    -- Distribution metrics
    messages_assigned BIGINT NOT NULL DEFAULT 0,
    messages_sent BIGINT NOT NULL DEFAULT 0,
    messages_failed BIGINT NOT NULL DEFAULT 0,
    
    -- Performance metrics
    average_processing_time_ms INTEGER,
    min_processing_time_ms INTEGER,
    max_processing_time_ms INTEGER,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_distribution_metrics_tenant (tenant_id, period_start DESC),
    INDEX idx_distribution_metrics_device (device_id, period_start DESC),
    UNIQUE INDEX idx_distribution_metrics_unique (tenant_id, device_id, period_start)
);

ALTER TABLE distribution_metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON distribution_metrics
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 7. Server Health Logs Table

```sql
CREATE TABLE server_health_logs (
    id BIGSERIAL PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES dedicated_servers(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Health check results
    check_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    response_time_ms INTEGER,
    status_code INTEGER,
    is_available BOOLEAN NOT NULL,
    
    -- Error information
    error_message TEXT,
    error_count INTEGER NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,2),
    
    -- Status classification
    health_status VARCHAR(20) NOT NULL, -- 'healthy', 'degraded', 'unhealthy'
    
    INDEX idx_server_health_logs_server (server_id, check_timestamp DESC),
    INDEX idx_server_health_logs_tenant (tenant_id, check_timestamp DESC),
    INDEX idx_server_health_logs_status (health_status)
);

ALTER TABLE server_health_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON server_health_logs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 8. Alerts Table

```sql
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id), -- NULL for system-wide alerts
    
    -- Alert information
    alert_type VARCHAR(50) NOT NULL, -- 'rating_change', 'server_failure', 'threshold_violation', 'device_failure'
    severity VARCHAR(20) NOT NULL, -- 'critical', 'warning', 'info'
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Alert context
    context JSONB, -- Additional context data
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'acknowledged', 'resolved'
    acknowledged_at TIMESTAMP,
    acknowledged_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP,
    
    -- Deduplication
    dedup_key VARCHAR(255), -- Prevents duplicate alerts within 5 minutes
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_alerts_tenant (tenant_id, created_at DESC),
    INDEX idx_alerts_type (alert_type),
    INDEX idx_alerts_severity (severity),
    INDEX idx_alerts_status (status),
    INDEX idx_alerts_dedup (dedup_key, created_at),
    
    CONSTRAINT check_severity CHECK (severity IN ('critical', 'warning', 'info')),
    CONSTRAINT check_status CHECK (status IN ('active', 'acknowledged', 'resolved'))
);

-- Partial index for active alerts only
CREATE INDEX idx_alerts_active ON alerts(tenant_id, created_at DESC) WHERE status = 'active';
```


#### 9. Signal/Push System Tables

**Message Queue Table**:
```sql
CREATE TABLE message_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Message information
    message_id VARCHAR(255) NOT NULL,
    message_type VARCHAR(10) NOT NULL, -- 'signal', 'push'
    message_content TEXT NOT NULL,
    recipient VARCHAR(50) NOT NULL,
    
    -- Routing information
    device_id UUID REFERENCES phone_devices(id),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'queued', -- 'queued', 'processing', 'sent', 'failed'
    priority INTEGER NOT NULL DEFAULT 1,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    
    -- Retry information
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    
    -- Metadata
    metadata JSONB,
    
    INDEX idx_message_queue_tenant_status (tenant_id, status, created_at),
    INDEX idx_message_queue_device (device_id, status),
    INDEX idx_message_queue_priority (priority DESC, created_at ASC)
);

ALTER TABLE message_queue ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON message_queue
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Waiting Messages Table**:
```sql
CREATE TABLE waiting_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message_id VARCHAR(255) NOT NULL UNIQUE,
    
    -- Status
    status VARCHAR(30) NOT NULL DEFAULT 'waiting_delivery_report',
    
    -- Timestamps
    sent_to_device_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_waiting_messages_tenant (tenant_id, created_at DESC),
    INDEX idx_waiting_messages_message_id (message_id)
);

ALTER TABLE waiting_messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON waiting_messages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Timing Violations Table**:
```sql
CREATE TABLE timing_violations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message_id VARCHAR(255) NOT NULL,
    message_type VARCHAR(10) NOT NULL,
    
    -- Timing information
    expected_gap_ms INTEGER NOT NULL,
    actual_gap_ms INTEGER NOT NULL,
    violation_severity VARCHAR(20) NOT NULL, -- 'minor', 'moderate', 'severe'
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_timing_violations_tenant (tenant_id, created_at DESC),
    INDEX idx_timing_violations_severity (violation_severity)
);

ALTER TABLE timing_violations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON timing_violations
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```


**Tenant Statistics Table**:
```sql
CREATE TABLE tenant_statistics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message_type VARCHAR(10) NOT NULL,
    
    -- Current metrics
    current_mpm DECIMAL(10,2) NOT NULL,
    current_stddev DECIMAL(10,2) NOT NULL,
    
    -- Historical baselines
    daily_avg_mpm DECIMAL(10,2) NOT NULL,
    daily_peak_mpm DECIMAL(10,2) NOT NULL,
    weekly_avg_mpm DECIMAL(10,2) NOT NULL,
    weekly_peak_mpm DECIMAL(10,2) NOT NULL,
    
    -- Anomaly detection
    anomaly_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    anomaly_level INTEGER NOT NULL DEFAULT 0,
    is_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
    anomaly_reason TEXT,
    
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, message_type),
    INDEX idx_tenant_statistics_anomaly (is_anomaly, anomaly_level)
);

ALTER TABLE tenant_statistics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tenant_statistics
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Enforcement Actions Table**:
```sql
CREATE TABLE enforcement_actions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Action information
    action_type VARCHAR(50) NOT NULL, -- 'warning', 'force_push', 'restore_signal'
    reason TEXT NOT NULL,
    anomaly_level INTEGER NOT NULL,
    
    -- Metrics at time of action
    violation_rate DECIMAL(5,2),
    z_score DECIMAL(5,2),
    peak_ratio DECIMAL(5,2),
    
    -- Action details
    previous_mode VARCHAR(10),
    new_mode VARCHAR(10),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    INDEX idx_enforcement_actions_tenant (tenant_id, created_at DESC),
    INDEX idx_enforcement_actions_type (action_type)
);

ALTER TABLE enforcement_actions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON enforcement_actions
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Tenant Timing Config Table**:
```sql
CREATE TABLE tenant_timing_config (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    
    -- Timing constraints
    signal_min_gap_ms INTEGER NOT NULL DEFAULT 2000,
    push_min_gap_ms INTEGER NOT NULL DEFAULT 500,
    
    -- Enforcement thresholds
    violation_rate_threshold DECIMAL(5,2) NOT NULL DEFAULT 50.0,
    z_score_threshold DECIMAL(5,2) NOT NULL DEFAULT 3.0,
    peak_ratio_threshold DECIMAL(5,2) NOT NULL DEFAULT 2.0,
    anomaly_duration_minutes INTEGER NOT NULL DEFAULT 10,
    
    -- Current mode
    current_mode VARCHAR(10) NOT NULL DEFAULT 'signal',
    forced_mode BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE tenant_timing_config ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tenant_timing_config
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Rating Calculation Correctness

*For any* set of performance metrics (success rate, sending volume, payment compliance, service quality), the calculated rating should equal the weighted sum of normalized components using the configured weights.

**Validates: Requirements 1.1**

### Property 2: Rating Category Classification

*For any* rating value between 0 and 100, the classification function should return: "excellent" for ratings ≥ 90, "good" for ratings 75-89, "average" for ratings 60-74, and "poor" for ratings < 60.

**Validates: Requirements 1.4**

### Property 3: Historical Data Retention

*For any* company rating record, if the record is less than 12 months old, it should not be deleted from the database.

**Validates: Requirements 1.6**

### Property 4: Rating Filter Correctness

*For any* filter criteria (category, period, company name), the query results should contain only records that match all specified criteria.

**Validates: Requirements 1.7**

### Property 5: Dedicated Server Routing

*For any* company with a dedicated server configured, all messages for that company should be routed to the dedicated server endpoint, never to shared infrastructure.

**Validates: Requirements 2.1**

### Property 6: Company Type Detection

*For any* company configuration, the routing engine should correctly identify whether the company has dedicated or shared routing based on the presence of a dedicated_server_id.

**Validates: Requirements 2.2**

### Property 7: Routing Isolation

*For any* two different companies, routing decisions for one company should not affect routing decisions for the other company.

**Validates: Requirements 2.3**

### Property 8: Unavailable Server Error Handling

*For any* message routed to an unavailable dedicated server, the routing engine should return an error and log the failure, never silently failing or routing to alternative infrastructure.

**Validates: Requirements 2.4**

### Property 9: Routing Metrics Tracking

*For any* routing operation, the routing metrics (request count, success/failure counts) should be incremented correctly for the target server.

**Validates: Requirements 2.8**

### Property 10: Distribution Strategy Execution

*For any* company with a configured distribution strategy, messages should be distributed to devices according to the strategy's algorithm (round robin, load balancing, priority, or custom rules).

**Validates: Requirements 3.1**

### Property 11: Round Robin Fairness

*For any* sequence of 1000 messages distributed using round robin strategy across N devices, each device should receive between (1000/N - 50) and (1000/N + 50) messages (5% variance).

**Validates: Requirements 3.2, 3.10**

### Property 12: Load Balancing Selection

*For any* device pool with varying loads, the load balancing strategy should always select the device with the minimum current load.

**Validates: Requirements 3.3**

### Property 13: Priority-Based Selection

*For any* device pool with varying priorities, the priority-based strategy should always select the device with the highest priority among available devices, and among devices with equal priority, select the one with lowest load.

**Validates: Requirements 3.4**

### Property 14: Device Failure Exclusion

*For any* device that fails to process a message, the device should be excluded from distribution for exactly 5 minutes, then automatically re-included.

**Validates: Requirements 3.8**

### Property 15: Distribution Metrics Tracking

*For any* message distribution operation, the device metrics (message count, success/failure counts) should be updated correctly for the selected device.

**Validates: Requirements 3.7**


### Property 16: Success Rate Calculation

*For any* performance metrics with total_messages > 0, the success rate should equal (successful_messages / total_messages) × 100.

**Validates: Requirements 4.1**

### Property 17: Volume Aggregation

*For any* time period, the sending volume should equal the count of all messages sent during that period.

**Validates: Requirements 4.2**

### Property 18: Payment Compliance Calculation

*For any* payment metrics with total_payments > 0, the payment compliance should equal (on_time_payments / total_payments) × 100.

**Validates: Requirements 4.3**

### Property 19: Time Range Filtering

*For any* time range filter (start_time, end_time), the query results should contain only records where the timestamp is >= start_time AND <= end_time.

**Validates: Requirements 4.6**

### Property 20: Threshold Alert Triggering

*For any* performance metric that falls below its configured threshold, an alert should be generated with appropriate severity level.

**Validates: Requirements 4.8**

### Property 21: Server Health Classification

*For any* health check result, the server status should be classified as: "healthy" if response_time < 200ms AND error_rate < 1%, "degraded" if response_time 200-500ms OR error_rate 1-5%, "unhealthy" if response_time > 500ms OR error_rate > 5%.

**Validates: Requirements 5.4**

### Property 22: Health History Retention

*For any* health check record, if the record is less than 30 days old, it should not be deleted from the database.

**Validates: Requirements 5.7**

### Property 23: Event-Driven Alert Generation

*For any* monitored event (rating change, server failure, device failure), an alert should be generated when the event occurs.

**Validates: Requirements 6.1, 6.2, 6.4**

### Property 24: Threshold-Based Alert Generation

*For any* metric that crosses a configured threshold, an alert should be generated with severity level matching the threshold configuration.

**Validates: Requirements 6.3**

### Property 25: Multi-Channel Alert Delivery

*For any* alert with multiple notification channels configured, the alert should be sent to all configured channels (email, SMS, in-app).

**Validates: Requirements 6.5**

### Property 26: Alert Deduplication

*For any* two identical events occurring within 5 minutes, only one alert should be generated (using dedup_key).

**Validates: Requirements 6.8**

### Property 27: Alert Filtering

*For any* alert filter criteria (severity, type, time range), the query results should contain only alerts matching all specified criteria.

**Validates: Requirements 6.9**

### Property 28: API Authentication Enforcement

*For any* API request without a valid API key, the request should be rejected with 401 Unauthorized status.

**Validates: Requirements 8.6**

### Property 29: API Rate Limiting

*For any* API key that exceeds 1000 requests per hour, subsequent requests should be rejected with 429 Too Many Requests status.

**Validates: Requirements 8.7**

### Property 30: JSON Response Format

*For any* API response, the response body should be valid JSON that can be parsed without errors.

**Validates: Requirements 8.8**

### Property 31: Error Response Completeness

*For any* API error response, the response should include an error code and error message.

**Validates: Requirements 8.9**

### Property 32: API Request Logging

*For any* API request, an audit log entry should be created with request details (endpoint, method, API key, timestamp).

**Validates: Requirements 8.10**

### Property 33: Input Validation

*For any* invalid input data (null required fields, out-of-range values, invalid formats), the system should reject the input and return a validation error.

**Validates: Requirements 10.1**

### Property 34: Transaction Atomicity

*For any* business operation that modifies multiple records, either all changes should be committed or all changes should be rolled back (no partial updates).

**Validates: Requirements 10.2**

### Property 35: Cache Consistency

*For any* cached data, if the cache entry is less than 5 minutes old, it should match the database value; if older than 5 minutes, it should be refreshed from the database.

**Validates: Requirements 10.3**

### Property 36: Concurrent Operation Safety

*For any* two concurrent operations on the same resource, the final state should be consistent and reflect both operations without data loss or corruption.

**Validates: Requirements 10.4**

### Property 37: Retry Logic Execution

*For any* transient failure, the system should retry the operation up to 3 times with exponential backoff before marking it as permanently failed.

**Validates: Requirements 10.5**

### Property 38: Operation Logging

*For any* significant business operation (rating calculation, routing decision, alert generation), a log entry should be created with operation details.

**Validates: Requirements 10.6**

### Property 39: Graceful Degradation

*For any* dependent service failure, the system should continue operating in degraded mode rather than crashing, returning appropriate error messages to users.

**Validates: Requirements 10.7**

### Property 40: Business Rule Validation

*For any* configuration that violates business rules (e.g., distribution percentages not summing to 100%, negative values), the system should reject the configuration with a clear error message.

**Validates: Requirements 10.8**


## Error Handling

### Error Categories

#### 1. Validation Errors
- Invalid input data (missing required fields, out-of-range values)
- Business rule violations (invalid configurations)
- **Response**: 400 Bad Request with detailed validation errors
- **Action**: Return immediately, do not process

#### 2. Authentication/Authorization Errors
- Missing or invalid API key
- Insufficient permissions for operation
- **Response**: 401 Unauthorized or 403 Forbidden
- **Action**: Log security event, return immediately

#### 3. Resource Not Found Errors
- Company, server, device, or alert not found
- **Response**: 404 Not Found
- **Action**: Return immediately with clear message

#### 4. Conflict Errors
- Duplicate configuration
- Concurrent modification conflicts
- **Response**: 409 Conflict
- **Action**: Suggest retry or manual resolution

#### 5. Rate Limiting Errors
- API rate limit exceeded
- **Response**: 429 Too Many Requests with Retry-After header
- **Action**: Log rate limit violation

#### 6. External Service Errors
- Dedicated server unavailable
- Database connection failure
- Redis connection failure
- **Response**: 503 Service Unavailable
- **Action**: Log error, trigger alert, retry with exponential backoff

#### 7. Internal Server Errors
- Unexpected exceptions
- Programming errors
- **Response**: 500 Internal Server Error
- **Action**: Log full stack trace, trigger critical alert

### Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "ERROR_CODE",
        "message": "Human-readable error message",
        "details": {
            "field": "Additional context"
        },
        "timestamp": "2024-01-01T12:00:00Z",
        "request_id": "uuid"
    }
}
```

### Retry Strategy

```go
type RetryConfig struct {
    MaxRetries      int           // 3
    InitialDelay    time.Duration // 100ms
    MaxDelay        time.Duration // 10s
    Multiplier      float64       // 2.0
    RetryableErrors []string      // List of retryable error codes
}

func (rc *RetryConfig) ShouldRetry(err error, attempt int) bool {
    if attempt >= rc.MaxRetries {
        return false
    }
    
    // Check if error is retryable
    for _, retryableErr := range rc.RetryableErrors {
        if strings.Contains(err.Error(), retryableErr) {
            return true
        }
    }
    
    return false
}

func (rc *RetryConfig) GetDelay(attempt int) time.Duration {
    delay := time.Duration(float64(rc.InitialDelay) * math.Pow(rc.Multiplier, float64(attempt)))
    if delay > rc.MaxDelay {
        delay = rc.MaxDelay
    }
    return delay
}
```

### Circuit Breaker

```go
type CircuitBreaker struct {
    FailureThreshold  int           // 10 failures
    SuccessThreshold  int           // 5 successes
    Timeout           time.Duration // 30 seconds
    State             CircuitState  // Open, HalfOpen, Closed
}

// Prevents cascading failures by stopping requests to failing services
```


## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests for specific scenarios and property-based tests for comprehensive coverage:

- **Unit Tests**: Verify specific examples, edge cases, error conditions, and integration points
- **Property Tests**: Verify universal properties across all inputs using randomized testing
- **Together**: Provide comprehensive coverage where unit tests catch concrete bugs and property tests verify general correctness

### Property-Based Testing

**Library**: Use `gopter` for Go property-based testing

**Configuration**:
- Minimum 100 iterations per property test
- Each test tagged with: `Feature: admin-interface-enhancements, Property {number}: {property_text}`
- Generators for: companies, servers, devices, messages, metrics, alerts

**Example Property Test**:
```go
// Feature: admin-interface-enhancements, Property 2: Rating Category Classification
func TestRatingCategoryClassification(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("rating classification is correct for all values", 
        prop.ForAll(
            func(rating float64) bool {
                category := ClassifyRating(rating)
                
                switch {
                case rating >= 90:
                    return category == "excellent"
                case rating >= 75:
                    return category == "good"
                case rating >= 60:
                    return category == "average"
                default:
                    return category == "poor"
                }
            },
            gen.Float64Range(0, 100),
        ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Unit Testing Focus Areas

1. **Specific Examples**:
   - Rating calculation with known inputs
   - Server health classification with specific response times
   - Alert generation for specific events

2. **Edge Cases**:
   - Empty device pools
   - Zero message counts
   - Boundary values for thresholds
   - Concurrent access to shared resources

3. **Error Conditions**:
   - Invalid API keys
   - Unavailable servers
   - Database connection failures
   - Malformed input data

4. **Integration Points**:
   - Tariff system integration
   - Gateway management integration
   - Balance management integration
   - Notification service integration

### Integration Testing

**Scenarios**:
1. End-to-end company rating calculation and alert generation
2. Message routing through dedicated server with health monitoring
3. Multi-device distribution with failure handling
4. Signal/Push mode switching based on timing violations
5. API authentication and rate limiting
6. Dashboard data aggregation from multiple sources

### Performance Testing

**Load Tests**:
- 100 concurrent admin users
- 1000 companies with real-time metrics
- 10,000 messages/minute through distribution system
- 100 API requests/second

**Benchmarks**:
- Rating calculation: < 10ms
- Routing decision: < 50ms (95th percentile)
- Health check: < 200ms
- API response: < 200ms average

### Security Testing

**Test Cases**:
1. Unauthenticated API access attempts
2. Cross-tenant data access attempts
3. SQL injection attempts in filters
4. Rate limit bypass attempts
5. Privilege escalation attempts

### Test Data Generators

```go
// Company generator
func GenCompany() gopter.Gen {
    return gopter.CombineGens(
        gen.UUID(),
        gen.AlphaString(),
        gen.Float64Range(0, 100),
        gen.IntRange(0, 1000000),
    ).Map(func(vals []interface{}) *Company {
        return &Company{
            ID:              vals[0].(uuid.UUID),
            Name:            vals[1].(string),
            Rating:          vals[2].(float64),
            TotalMessages:   vals[3].(int),
        }
    })
}

// Performance metrics generator
func GenPerformanceMetrics() gopter.Gen {
    return gopter.CombineGens(
        gen.IntRange(0, 1000000),
        gen.IntRange(0, 1000000),
    ).Map(func(vals []interface{}) *PerformanceMetrics {
        total := vals[0].(int)
        successful := vals[1].(int)
        if successful > total {
            successful = total
        }
        return &PerformanceMetrics{
            TotalMessages:      total,
            SuccessfulMessages: successful,
        }
    })
}

// Device pool generator
func GenDevicePool() gopter.Gen {
    return gen.SliceOf(GenDevice(), reflect.TypeOf(&Device{}))
}
```

### Continuous Integration

**CI Pipeline**:
1. Run unit tests on every commit
2. Run property tests on every PR
3. Run integration tests on merge to main
4. Run performance tests nightly
5. Run security tests weekly

**Coverage Requirements**:
- Unit test coverage: > 80%
- Property test coverage: All correctness properties
- Integration test coverage: All critical paths
- API endpoint coverage: 100%


## API Endpoints

### Company Rating APIs

#### GET /api/admin/companies/{tenant_id}/rating
Get current rating for a company

**Response**:
```json
{
    "tenant_id": "uuid",
    "overall_rating": 85.5,
    "rating_category": "good",
    "components": {
        "success_rate": 92.3,
        "sending_volume": 75000,
        "payment_compliance": 100.0,
        "service_quality_score": 88.0
    },
    "calculation_timestamp": "2024-01-01T12:00:00Z"
}
```

#### GET /api/admin/companies/ratings
List all company ratings with filtering

**Query Parameters**:
- `category`: Filter by rating category
- `min_rating`: Minimum rating value
- `max_rating`: Maximum rating value
- `period_start`: Start of time range
- `period_end`: End of time range
- `page`: Page number
- `limit`: Results per page

#### GET /api/admin/companies/{tenant_id}/rating/history
Get rating history for a company

**Query Parameters**:
- `period_start`: Start date (default: 12 months ago)
- `period_end`: End date (default: now)

### Performance Metrics APIs

#### GET /api/admin/companies/{tenant_id}/metrics
Get performance metrics for a company

**Query Parameters**:
- `period_type`: hourly, daily, monthly
- `period_start`: Start of time range
- `period_end`: End of time range

**Response**:
```json
{
    "tenant_id": "uuid",
    "metrics": [
        {
            "period_start": "2024-01-01T00:00:00Z",
            "period_end": "2024-01-01T23:59:59Z",
            "total_messages": 10000,
            "successful_messages": 9500,
            "failed_messages": 500,
            "success_rate": 95.0,
            "average_delivery_time_ms": 1500,
            "error_rate": 5.0
        }
    ]
}
```

### Routing Configuration APIs

#### GET /api/admin/routing/{tenant_id}
Get routing configuration for a company

#### PUT /api/admin/routing/{tenant_id}
Update routing configuration

**Request Body**:
```json
{
    "routing_type": "dedicated",
    "dedicated_server_id": "uuid",
    "dedicated_server_url": "https://server.example.com",
    "dedicated_server_api_key": "encrypted_key"
}
```

#### GET /api/admin/routing/{tenant_id}/metrics
Get routing metrics for a company

### Distribution Strategy APIs

#### GET /api/admin/distribution/{tenant_id}
Get distribution strategy for a company

#### PUT /api/admin/distribution/{tenant_id}
Update distribution strategy

**Request Body**:
```json
{
    "strategy_type": "load_balancing",
    "strategy_config": {
        "load_threshold": 80,
        "rebalance_interval": 60
    }
}
```

#### GET /api/admin/distribution/{tenant_id}/devices
List devices for a company

#### GET /api/admin/distribution/{tenant_id}/metrics
Get distribution metrics per device

### Server Health APIs

#### GET /api/admin/servers
List all dedicated servers

#### GET /api/admin/servers/{server_id}
Get server details and current health status

**Response**:
```json
{
    "server_id": "uuid",
    "tenant_id": "uuid",
    "server_url": "https://server.example.com",
    "health_status": "healthy",
    "last_check": "2024-01-01T12:00:00Z",
    "response_time_ms": 150,
    "error_rate": 0.5,
    "metrics": {
        "total_requests": 100000,
        "successful_requests": 99500,
        "failed_requests": 500,
        "average_latency_ms": 145
    }
}
```

#### GET /api/admin/servers/{server_id}/health/history
Get health check history for a server

**Query Parameters**:
- `period_start`: Start date (default: 30 days ago)
- `period_end`: End date (default: now)

### Alert Management APIs

#### GET /api/admin/alerts
List alerts with filtering

**Query Parameters**:
- `tenant_id`: Filter by company
- `severity`: critical, warning, info
- `alert_type`: rating_change, server_failure, threshold_violation, device_failure
- `status`: active, acknowledged, resolved
- `period_start`: Start of time range
- `period_end`: End of time range

#### GET /api/admin/alerts/{alert_id}
Get alert details

#### PUT /api/admin/alerts/{alert_id}/acknowledge
Acknowledge an alert

#### PUT /api/admin/alerts/{alert_id}/resolve
Resolve an alert

#### POST /api/admin/alerts/config
Configure alert thresholds and notification preferences

**Request Body**:
```json
{
    "tenant_id": "uuid",
    "thresholds": {
        "success_rate_warning": 90.0,
        "success_rate_critical": 80.0,
        "response_time_warning": 500,
        "response_time_critical": 1000
    },
    "notification_channels": ["email", "sms", "in_app"],
    "notification_emails": ["admin@example.com"]
}
```

### Signal/Push System APIs

#### GET /api/admin/signalpush/{tenant_id}/config
Get Signal/Push timing configuration

#### PUT /api/admin/signalpush/{tenant_id}/config
Update timing configuration

**Request Body**:
```json
{
    "signal_min_gap_ms": 2000,
    "push_min_gap_ms": 500,
    "violation_rate_threshold": 50.0,
    "z_score_threshold": 3.0,
    "current_mode": "signal"
}
```

#### GET /api/admin/signalpush/{tenant_id}/statistics
Get timing statistics and anomaly detection results

**Response**:
```json
{
    "tenant_id": "uuid",
    "current_mode": "signal",
    "forced_mode": false,
    "statistics": {
        "current_mpm": 120.5,
        "daily_avg_mpm": 100.0,
        "daily_peak_mpm": 150.0,
        "anomaly_level": 0,
        "is_anomaly": false
    },
    "violations": {
        "last_hour": 5,
        "last_24_hours": 50,
        "violation_rate": 2.5
    }
}
```

#### GET /api/admin/signalpush/{tenant_id}/violations
Get timing violation history

#### POST /api/admin/signalpush/{tenant_id}/force-mode
Manually force Signal or Push mode

**Request Body**:
```json
{
    "mode": "push",
    "reason": "Manual override for testing"
}
```

### Dashboard Statistics APIs

#### GET /api/admin/dashboard/overview
Get system-wide statistics

**Response**:
```json
{
    "total_companies": 1000,
    "active_companies": 950,
    "total_messages_today": 1000000,
    "success_rate_today": 95.5,
    "rating_distribution": {
        "excellent": 250,
        "good": 500,
        "average": 150,
        "poor": 50
    },
    "active_alerts": 15,
    "critical_alerts": 2,
    "server_health": {
        "healthy": 45,
        "degraded": 3,
        "unhealthy": 2
    }
}
```


## Integration with Existing Systems

### 1. Tariff Pricing System Integration

**Integration Points**:
- **Balance Information**: Retrieve company balance for payment compliance calculation
- **Transaction History**: Access billing transactions for payment tracking
- **Subscription Data**: Get subscription status for service quality scoring

**Implementation**:
```go
type TariffSystemClient struct {
    baseURL string
    apiKey  string
}

func (tsc *TariffSystemClient) GetCompanyBalance(tenantID uuid.UUID) (float64, error) {
    // Call tariff system API
    resp, err := http.Get(fmt.Sprintf("%s/api/balance/%s", tsc.baseURL, tenantID))
    // Parse and return balance
}

func (tsc *TariffSystemClient) GetPaymentHistory(tenantID uuid.UUID, period time.Duration) ([]*Payment, error) {
    // Retrieve payment history for compliance calculation
}
```

### 2. Dedicated Gateways Integration

**Integration Points**:
- **Gateway Registry**: Access dedicated gateway configurations
- **Gateway Statistics**: Retrieve message statistics per gateway
- **Health Monitoring**: Monitor gateway availability and performance

**Implementation**:
```go
type GatewayManagerClient struct {
    db *gorm.DB
}

func (gmc *GatewayManagerClient) GetDedicatedGateway(tenantID uuid.UUID) (*DedicatedGateway, error) {
    var gateway DedicatedGateway
    err := gmc.db.Where("tenant_id = ?", tenantID).First(&gateway).Error
    return &gateway, err
}

func (gmc *GatewayManagerClient) GetGatewayStatistics(gatewayID uuid.UUID, period time.Duration) (*GatewayStats, error) {
    // Query gateway_statistics table
}
```

### 3. Resource Management Integration

**Integration Points**:
- **Resource Quotas**: Check resource limits for companies
- **Resource Usage**: Track actual resource consumption
- **Quota Enforcement**: Integrate with resource limiting

**Implementation**:
```go
type ResourceManagerClient struct {
    db *gorm.DB
}

func (rmc *ResourceManagerClient) GetResourceUsage(tenantID uuid.UUID) (*ResourceUsage, error) {
    var usage ResourceUsage
    err := rmc.db.Where("tenant_id = ?", tenantID).First(&usage).Error
    return &usage, err
}

func (rmc *ResourceManagerClient) CheckQuotaCompliance(tenantID uuid.UUID) (bool, error) {
    // Compare usage against quotas
}
```

### 4. Message Lifecycle System Integration

**Integration Points**:
- **Message Statistics**: Access message success/failure counts
- **Delivery Reports**: Retrieve delivery time metrics
- **Worker Pool**: Integrate with existing worker infrastructure

**Implementation**:
```go
type MessageSystemClient struct {
    db *gorm.DB
}

func (msc *MessageSystemClient) GetMessageStatistics(tenantID uuid.UUID, period time.Duration) (*MessageStats, error) {
    // Query messages table for statistics
    var stats MessageStats
    err := msc.db.Raw(`
        SELECT 
            COUNT(*) as total_messages,
            SUM(CASE WHEN status = 'delivered' THEN 1 ELSE 0 END) as successful_messages,
            SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_messages,
            AVG(EXTRACT(EPOCH FROM (delivered_at - created_at)) * 1000) as avg_delivery_time_ms
        FROM messages
        WHERE tenant_id = ? AND created_at >= ?
    `, tenantID, time.Now().Add(-period)).Scan(&stats).Error
    
    return &stats, err
}
```

### 5. WebSocket Hub Integration

**Integration Points**:
- **Device Connection Status**: Check if devices are online
- **Message Sending**: Send messages through WebSocket
- **Connection Metrics**: Track connection stability

**Implementation**:
```go
type WebSocketHubClient struct {
    hub *ws.Hub
}

func (wsc *WebSocketHubClient) IsDeviceOnline(deviceID uuid.UUID) bool {
    return wsc.hub.IsClientConnected(deviceID.String())
}

func (wsc *WebSocketHubClient) SendMessage(deviceID uuid.UUID, message *Message) error {
    return wsc.hub.SendToClient(deviceID.String(), message)
}

func (wsc *WebSocketHubClient) GetDeviceMetrics(deviceID uuid.UUID) (*DeviceMetrics, error) {
    // Get connection uptime, message count, etc.
}
```

### 6. Notification Service Integration

**Integration Points**:
- **Alert Notifications**: Send alerts via email, SMS, in-app
- **Notification Templates**: Use existing templates
- **Delivery Tracking**: Track notification delivery status

**Implementation**:
```go
type NotificationServiceClient struct {
    service *notifications.Service
}

func (nsc *NotificationServiceClient) SendAlert(alert *Alert, channels []string) error {
    for _, channel := range channels {
        switch channel {
        case "email":
            err := nsc.service.SendEmail(alert.ToEmailNotification())
            if err != nil {
                return err
            }
        case "sms":
            err := nsc.service.SendSMS(alert.ToSMSNotification())
            if err != nil {
                return err
            }
        case "in_app":
            err := nsc.service.SendInApp(alert.ToInAppNotification())
            if err != nil {
                return err
            }
        }
    }
    return nil
}
```

### 7. Monitoring and Metrics Integration

**Integration Points**:
- **Prometheus Metrics**: Export admin system metrics
- **Grafana Dashboards**: Create admin-specific dashboards
- **Alert Manager**: Integrate with existing alerting

**Prometheus Metrics**:
```go
var (
    companyRatingsCalculated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "admin_company_ratings_calculated_total",
            Help: "Total number of company ratings calculated",
        },
        []string{"category"},
    )
    
    serverHealthChecks = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "admin_server_health_checks_total",
            Help: "Total number of server health checks performed",
        },
        []string{"server_id", "status"},
    )
    
    messageDistributions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "admin_message_distributions_total",
            Help: "Total number of messages distributed",
        },
        []string{"tenant_id", "strategy", "device_id"},
    )
    
    timingViolations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "admin_timing_violations_total",
            Help: "Total number of timing violations detected",
        },
        []string{"tenant_id", "severity"},
    )
    
    alertsGenerated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "admin_alerts_generated_total",
            Help: "Total number of alerts generated",
        },
        []string{"type", "severity"},
    )
)
```


## Admin Dashboard UI Architecture

### Technology Stack

- **Framework**: React 18+ with TypeScript
- **State Management**: React Query for server state, Zustand for client state
- **UI Components**: Tailwind CSS + shadcn/ui
- **Charts**: Recharts for data visualization
- **Real-time Updates**: WebSocket for live metrics
- **Internationalization**: i18next with RTL support

### Page Structure

```
Admin Dashboard
├── Overview Page
│   ├── System Statistics Cards
│   ├── Rating Distribution Chart
│   ├── Active Alerts Panel
│   └── Server Health Summary
├── Companies Page
│   ├── Companies List Table
│   ├── Search and Filters
│   ├── Rating Category Filter
│   └── Bulk Actions
├── Company Details Page
│   ├── Company Info Header
│   ├── Rating History Chart
│   ├── Performance Metrics Tabs
│   ├── Devices List
│   ├── Routing Configuration
│   └── Distribution Strategy
├── Servers Page
│   ├── Servers List Table
│   ├── Health Status Filters
│   ├── Real-time Health Indicators
│   └── Server Details Modal
├── Distribution Page
│   ├── Strategy Configuration
│   ├── Device Pool Management
│   ├── Distribution Metrics
│   └── Load Visualization
├── Alerts Page
│   ├── Active Alerts List
│   ├── Alert History
│   ├── Severity Filters
│   └── Alert Configuration
└── Signal/Push Monitor Page
    ├── Timing Statistics
    ├── Violation Trends
    ├── Anomaly Detection Status
    └── Mode Override Controls
```

### Component Architecture

```typescript
// Company Rating Component
interface CompanyRatingProps {
    tenantId: string;
    showHistory?: boolean;
}

const CompanyRating: React.FC<CompanyRatingProps> = ({ tenantId, showHistory }) => {
    const { data: rating, isLoading } = useCompanyRating(tenantId);
    
    return (
        <div className="rating-card">
            <RatingBadge category={rating.category} />
            <RatingScore value={rating.overall_rating} />
            <RatingComponents components={rating.components} />
            {showHistory && <RatingHistoryChart tenantId={tenantId} />}
        </div>
    );
};

// Server Health Monitor Component
interface ServerHealthProps {
    serverId: string;
    realTime?: boolean;
}

const ServerHealthMonitor: React.FC<ServerHealthProps> = ({ serverId, realTime }) => {
    const { data: health } = useServerHealth(serverId);
    const { data: liveHealth } = useWebSocket(`/ws/servers/${serverId}/health`, {
        enabled: realTime
    });
    
    const currentHealth = realTime ? liveHealth : health;
    
    return (
        <div className="health-monitor">
            <HealthStatusIndicator status={currentHealth.status} />
            <MetricCard label="Response Time" value={currentHealth.response_time_ms} unit="ms" />
            <MetricCard label="Error Rate" value={currentHealth.error_rate} unit="%" />
            <HealthHistoryChart serverId={serverId} />
        </div>
    );
};

// Distribution Strategy Configurator
interface DistributionConfigProps {
    tenantId: string;
}

const DistributionConfigurator: React.FC<DistributionConfigProps> = ({ tenantId }) => {
    const { data: strategy } = useDistributionStrategy(tenantId);
    const { mutate: updateStrategy } = useUpdateDistributionStrategy();
    
    return (
        <div className="distribution-config">
            <StrategySelector 
                value={strategy.strategy_type}
                onChange={(type) => updateStrategy({ tenantId, strategy_type: type })}
            />
            <DevicePoolManager tenantId={tenantId} />
            <DistributionMetrics tenantId={tenantId} />
        </div>
    );
};
```

### Real-time Updates

```typescript
// WebSocket hook for real-time metrics
function useRealtimeMetrics(tenantId: string) {
    const [metrics, setMetrics] = useState<Metrics | null>(null);
    
    useEffect(() => {
        const ws = new WebSocket(`ws://api/admin/ws/metrics/${tenantId}`);
        
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            setMetrics(data);
        };
        
        return () => ws.close();
    }, [tenantId]);
    
    return metrics;
}

// Auto-refresh for non-WebSocket data
function useAutoRefresh<T>(
    queryKey: string[],
    queryFn: () => Promise<T>,
    interval: number = 30000
) {
    return useQuery(queryKey, queryFn, {
        refetchInterval: interval,
        refetchIntervalInBackground: true
    });
}
```

### Responsive Design

```typescript
// Breakpoints
const breakpoints = {
    mobile: '640px',
    tablet: '768px',
    desktop: '1024px',
    wide: '1280px'
};

// Responsive layout components
const ResponsiveGrid: React.FC = ({ children }) => {
    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {children}
        </div>
    );
};
```

### Internationalization

```typescript
// i18n configuration
const resources = {
    en: {
        translation: {
            'companies.rating.excellent': 'Excellent',
            'companies.rating.good': 'Good',
            'companies.rating.average': 'Average',
            'companies.rating.poor': 'Poor',
            'servers.health.healthy': 'Healthy',
            'servers.health.degraded': 'Degraded',
            'servers.health.unhealthy': 'Unhealthy'
        }
    },
    ar: {
        translation: {
            'companies.rating.excellent': 'ممتاز',
            'companies.rating.good': 'جيد',
            'companies.rating.average': 'متوسط',
            'companies.rating.poor': 'ضعيف',
            'servers.health.healthy': 'سليم',
            'servers.health.degraded': 'متدهور',
            'servers.health.unhealthy': 'غير سليم'
        }
    }
};

// RTL support
const direction = i18n.language === 'ar' ? 'rtl' : 'ltr';
document.documentElement.dir = direction;
```

### Performance Optimization

```typescript
// Virtualized lists for large datasets
import { FixedSizeList } from 'react-window';

const CompanyList: React.FC<{ companies: Company[] }> = ({ companies }) => {
    return (
        <FixedSizeList
            height={600}
            itemCount={companies.length}
            itemSize={80}
            width="100%"
        >
            {({ index, style }) => (
                <div style={style}>
                    <CompanyRow company={companies[index]} />
                </div>
            )}
        </FixedSizeList>
    );
};

// Lazy loading for charts
const RatingHistoryChart = lazy(() => import('./RatingHistoryChart'));

// Memoization for expensive calculations
const distributionStats = useMemo(() => {
    return calculateDistributionStats(messages, devices);
}, [messages, devices]);
```


## Security and Access Control

### Role-Based Access Control (RBAC)

**Roles**:
1. **Super Admin**: Full access to all features and all companies
2. **Admin**: Access to assigned companies and read-only system features
3. **Viewer**: Read-only access to assigned companies

**Permissions Matrix**:

| Feature | Super Admin | Admin | Viewer |
|---------|-------------|-------|--------|
| View all companies | ✓ | ✗ | ✗ |
| View assigned companies | ✓ | ✓ | ✓ |
| Modify company ratings | ✓ | ✗ | ✗ |
| Configure routing | ✓ | ✓ | ✗ |
| Configure distribution | ✓ | ✓ | ✗ |
| View server health | ✓ | ✓ | ✓ |
| Manage alerts | ✓ | ✓ | ✗ |
| Force Signal/Push mode | ✓ | ✗ | ✗ |
| View audit logs | ✓ | ✓ | ✗ |

### Authentication

```go
type AdminUser struct {
    ID       uuid.UUID
    Email    string
    Role     string // 'super_admin', 'admin', 'viewer'
    TenantIDs []uuid.UUID // Assigned companies (empty for super_admin)
}

func (au *AdminUser) CanAccessTenant(tenantID uuid.UUID) bool {
    if au.Role == "super_admin" {
        return true
    }
    
    for _, id := range au.TenantIDs {
        if id == tenantID {
            return true
        }
    }
    
    return false
}

func (au *AdminUser) CanModify() bool {
    return au.Role == "super_admin" || au.Role == "admin"
}
```

### Authorization Middleware

```go
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := GetCurrentUser(c)
        
        for _, role := range allowedRoles {
            if user.Role == role {
                c.Next()
                return
            }
        }
        
        c.JSON(http.StatusForbidden, gin.H{
            "error": "insufficient permissions"
        })
        c.Abort()
    }
}

func RequireTenantAccess() gin.HandlerFunc {
    return func(c *gin.Context) {
        user := GetCurrentUser(c)
        tenantID := c.Param("tenant_id")
        
        if !user.CanAccessTenant(uuid.MustParse(tenantID)) {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "access denied to this company"
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Audit Trail

```go
type AuditLog struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Action      string // 'view', 'create', 'update', 'delete'
    Resource    string // 'company_rating', 'routing_config', 'alert', etc.
    ResourceID  string
    TenantID    *uuid.UUID
    Changes     map[string]interface{} // Before/after values
    IPAddress   string
    UserAgent   string
    Timestamp   time.Time
}

func LogAuditEvent(ctx context.Context, event *AuditLog) error {
    return db.Create(event).Error
}
```

### Data Encryption

**At Rest**:
- Database encryption using PostgreSQL transparent data encryption
- Sensitive fields (API keys, credentials) encrypted using AES-256

**In Transit**:
- TLS 1.3 for all API communications
- WSS (WebSocket Secure) for real-time updates

### Input Sanitization

```go
func SanitizeInput(input string) string {
    // Remove SQL injection attempts
    input = strings.ReplaceAll(input, "'", "")
    input = strings.ReplaceAll(input, "\"", "")
    input = strings.ReplaceAll(input, ";", "")
    input = strings.ReplaceAll(input, "--", "")
    
    // Remove XSS attempts
    input = html.EscapeString(input)
    
    return input
}

func ValidateUUID(id string) error {
    _, err := uuid.Parse(id)
    return err
}
```

## Deployment and Operations

### Infrastructure Requirements

**Backend**:
- Go 1.21+
- 4 CPU cores minimum
- 8GB RAM minimum
- 100GB SSD storage

**Database**:
- PostgreSQL 15+
- 8 CPU cores
- 16GB RAM
- 500GB SSD storage
- Automated backups every 6 hours

**Cache**:
- Redis 7+
- 4GB RAM
- Persistence enabled (AOF + RDB)

**Monitoring**:
- Prometheus for metrics collection
- Grafana for visualization
- AlertManager for alerting

### Deployment Strategy

**Blue-Green Deployment**:
1. Deploy new version to green environment
2. Run smoke tests
3. Switch traffic to green
4. Monitor for issues
5. Keep blue as fallback for 24 hours

**Database Migrations**:
```bash
# Run migrations before deployment
./migrate up

# Rollback if needed
./migrate down 1
```

### Monitoring and Alerting

**Key Metrics**:
- API response time (p50, p95, p99)
- Rating calculation latency
- Server health check success rate
- Distribution throughput
- Alert generation rate
- Database query performance
- Cache hit rate

**Critical Alerts**:
- API response time > 1s for 5 minutes
- Database connection pool exhausted
- Redis connection failure
- Server health check failure rate > 10%
- Alert generation failure

### Backup and Recovery

**Backup Strategy**:
- Database: Full backup daily, incremental every 6 hours
- Configuration: Version controlled in Git
- Retention: 30 days for daily backups, 7 days for incremental

**Recovery Procedures**:
1. Identify failure point
2. Restore from latest backup
3. Replay transaction logs if available
4. Verify data integrity
5. Resume operations

### Scaling Strategy

**Horizontal Scaling**:
- Add more backend instances behind load balancer
- Use Redis Cluster for cache scaling
- PostgreSQL read replicas for read-heavy operations

**Vertical Scaling**:
- Increase CPU/RAM for backend instances
- Upgrade database instance size
- Increase Redis memory

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)
- Database schema creation
- Core data models
- Basic API endpoints
- Authentication and authorization

### Phase 2: Company Rating System (Weeks 3-4)
- Rating calculation engine
- Performance metrics collection
- Rating history management
- Alert generation for rating changes

### Phase 3: Server Health Monitoring (Weeks 5-6)
- Health check implementation
- Status classification
- Health history logging
- Server failure alerts

### Phase 4: Message Distribution (Weeks 7-8)
- Distribution strategy implementations
- Device pool management
- Load tracking
- Failure handling

### Phase 5: Signal/Push Integration (Weeks 9-11)
- Timing monitor
- Statistical analyzer
- Anomaly detector
- Auto-enforcement system
- Adaptive sender
- Delivery report processor

### Phase 6: Admin Dashboard UI (Weeks 12-14)
- Overview page
- Companies management
- Server monitoring
- Distribution configuration
- Alert management
- Signal/Push monitor

### Phase 7: Testing and Optimization (Weeks 15-16)
- Property-based tests
- Integration tests
- Performance testing
- Security testing
- Bug fixes and optimization

### Phase 8: Documentation and Deployment (Week 17)
- API documentation
- User guides
- Deployment procedures
- Training materials
- Production deployment

## Conclusion

This design provides a comprehensive architecture for admin interface enhancements with deep integration of the Signal/Push messaging routing system. The system is built on solid foundations of:

- **Scalability**: Supports 1000+ companies with real-time monitoring
- **Reliability**: Comprehensive error handling and retry mechanisms
- **Security**: Role-based access control and audit trails
- **Performance**: Optimized queries and caching strategies
- **Maintainability**: Clean architecture with clear separation of concerns
- **Testability**: Property-based testing for correctness guarantees

The integration with existing systems (tariff pricing, dedicated gateways, resource management) ensures seamless operation within the broader platform ecosystem.

