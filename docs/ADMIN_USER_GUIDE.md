# Admin Interface User Guide

**Version:** 1.0.0  
**Last Updated:** April 9, 2026

---

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Dashboard Overview](#dashboard-overview)
4. [Company Management](#company-management)
5. [Server Health Monitoring](#server-health-monitoring)
6. [Alert Management](#alert-management)
7. [Distribution Configuration](#distribution-configuration)
8. [Signal/Push Monitoring](#signalpush-monitoring)
9. [User Roles & Permissions](#user-roles--permissions)
10. [Troubleshooting](#troubleshooting)

---

## Introduction

The Admin Interface is a comprehensive management system for monitoring and controlling the Al-Hudhud SMS Gateway platform. It provides real-time insights into company performance, server health, message distribution, and system alerts.

### Key Features

- **Company Rating System**: Automated rating based on performance metrics
- **Server Health Monitoring**: Real-time health checks for dedicated servers
- **Smart Distribution**: Multiple strategies for message distribution across devices
- **Signal/Push Monitoring**: Timing compliance and anomaly detection
- **Alert Management**: Configurable alerts with multiple notification channels
- **Routing Configuration**: Dedicated vs shared server routing

---

## Getting Started

### Logging In

1. Navigate to the admin interface URL
2. Enter your credentials (username and password)
3. Click "Login"

### First-Time Setup

After logging in for the first time:

1. Review the dashboard overview
2. Check active alerts
3. Verify server health status
4. Review company ratings

### Navigation

The sidebar menu provides access to all major sections:

- **Dashboard**: System overview and statistics
- **Overview**: Detailed system metrics
- **Companies**: Company ratings and performance
- **Servers**: Server health monitoring
- **Alerts**: Alert management
- **Distribution**: Message distribution configuration
- **Signal/Push**: Timing monitoring and enforcement

---

## Dashboard Overview

The dashboard provides a high-level view of the entire system.

### Statistics Cards

- **Total Companies**: Number of active companies
- **Active Servers**: Number of healthy dedicated servers
- **Messages Today**: Total messages processed today
- **Success Rate**: Overall system success rate

### Rating Distribution Chart

Visual representation of companies by rating category:
- Excellent (90-100)
- Good (75-90)
- Average (60-75)
- Poor (0-60)

### Active Alerts Panel

Shows critical and warning alerts that require attention.

### Server Health Summary

Quick overview of server health status:
- Healthy (green)
- Degraded (yellow)
- Unhealthy (red)

---

## Company Management

### Viewing Company Ratings

1. Click "Companies" in the sidebar
2. View the list of all companies with their ratings
3. Use filters to narrow down results:
   - Rating category
   - Date range
   - Search by name

### Company Details

Click on any company to view detailed information:

#### Rating Tab
- Overall rating score
- Rating category
- Component breakdown:
  - Success rate
  - Sending volume
  - Payment compliance
  - Service quality score

#### Performance Tab
- Historical performance metrics
- Success/failure trends
- Message volume over time

#### History Tab
- Rating history chart
- Historical data points
- Trend analysis

#### Routing Tab
- Current routing configuration
- Dedicated server assignment (if applicable)
- Routing metrics

### Understanding Ratings

**Rating Components:**

1. **Success Rate (40% weight)**
   - Percentage of successfully delivered messages
   - Target: > 95%

2. **Sending Volume (20% weight)**
   - Total messages sent
   - Normalized against system average

3. **Payment Compliance (25% weight)**
   - Percentage of payments made on time
   - Target: 100%

4. **Service Quality (15% weight)**
   - Composite score based on various factors
   - Includes response time, error handling, etc.

**Rating Categories:**

- **Excellent**: 90-100 points
- **Good**: 75-89 points
- **Average**: 60-74 points
- **Poor**: 0-59 points

---

## Server Health Monitoring

### Server List

View all dedicated servers with their current health status.

**Health Status Indicators:**

- 🟢 **Healthy**: Response time < 200ms AND error rate < 1%
- 🟡 **Degraded**: Response time 200-500ms OR error rate 1-5%
- 🔴 **Unhealthy**: Response time > 500ms OR error rate > 5%

### Server Details

Click on any server to view:

- Current health metrics
- Response time trends
- Error rate history
- Uptime percentage
- Last health check timestamp

### Health History Chart

Visual representation of server health over time (up to 30 days).

### Automatic Health Checks

- Health checks run every 30 seconds
- Alerts triggered on status changes
- Historical data retained for 30 days

---

## Alert Management

### Viewing Alerts

1. Click "Alerts" in the sidebar
2. View active, acknowledged, and resolved alerts
3. Filter by:
   - Status (active, acknowledged, resolved)
   - Severity (critical, warning, info)
   - Type
   - Date range

### Alert Types

**Rating Alerts:**
- Rating degraded
- Rating improved
- Rating threshold crossed

**Server Health Alerts:**
- Server unhealthy
- Server degraded
- Server recovered

**Distribution Alerts:**
- Device failure
- High load
- Distribution imbalance

**Signal/Push Alerts:**
- Timing violations
- Anomaly detected
- Mode forced

### Managing Alerts

**Acknowledge Alert:**
1. Click on an alert
2. Click "Acknowledge"
3. Alert status changes to "acknowledged"

**Resolve Alert:**
1. Click on an alert
2. Click "Resolve"
3. Alert status changes to "resolved"

### Alert Configuration

**Super Admins** can configure alert thresholds:

1. Click "Alerts" → "Configuration"
2. Set thresholds for:
   - Response time (warning & critical)
   - Error rate (warning & critical)
   - Success rate (warning & critical)
3. Configure notification channels:
   - Email notifications
   - SMS notifications
   - In-app notifications
4. Click "Save Configuration"

---

## Distribution Configuration

### Distribution Strategies

Four strategies available for message distribution:

#### 1. Round Robin
- Distributes messages evenly across all devices
- Best for: Equal load distribution
- Configuration: None required

#### 2. Load Balancing
- Selects device with lowest current load
- Best for: Optimal resource utilization
- Configuration: Max load per device (optional)

#### 3. Priority-Based
- Distributes based on device priority levels
- Best for: Prioritizing specific devices
- Configuration: Device priorities

#### 4. Custom Rules
- Uses custom JSON rules for distribution
- Best for: Complex distribution logic
- Configuration: Custom rules JSON

### Configuring Distribution

1. Navigate to company details
2. Click "Distribution" tab
3. Select strategy type
4. Configure strategy parameters (if applicable)
5. Click "Save Configuration"

### Viewing Distribution Metrics

- Total messages distributed
- Messages per device
- Distribution fairness score
- Device utilization

### Device Management

View and manage devices:
- Device status (active, inactive, failed)
- Current load
- Total messages sent
- Success/failure rates

---

## Signal/Push Monitoring

### Understanding Signal vs Push

**Signal Mode:**
- Minimum gap: 5000ms between messages
- More reliable delivery
- Slower throughput

**Push Mode:**
- Minimum gap: 1000ms between messages
- Faster throughput
- May have lower reliability

### Timing Statistics

View real-time statistics:
- Current MPM (Messages Per Minute)
- Daily average MPM
- Daily peak MPM
- Standard deviation

### Anomaly Detection

System automatically detects anomalies using:
- **Z-Score**: Measures deviation from average
- **Peak Ratio**: Compares current to peak
- **Violation Rate**: Percentage of timing violations

**Anomaly Levels:**
- **Minor**: Log only
- **Moderate**: Send warning
- **Severe**: Force Push mode

### Viewing Violations

1. Click "Signal/Push" in sidebar
2. Select company
3. View violations list with:
   - Expected gap
   - Actual gap
   - Severity
   - Timestamp

### Manual Mode Control

**Super Admins** can manually force mode:

1. Navigate to Signal/Push monitor
2. Click "Force Signal Mode" or "Force Push Mode"
3. Confirm action
4. System switches mode immediately

**Note:** Manual mode changes should be used sparingly and only when necessary.

---

## User Roles & Permissions

### Super Admin

**Full Access:**
- View all data
- Modify all configurations
- Manage alerts
- Force mode changes
- Update routing
- Configure distribution

### Staff

**Read-Only Access:**
- View dashboard
- View company ratings
- View server health
- View alerts
- View distribution config
- View Signal/Push stats

**Cannot:**
- Modify configurations
- Force mode changes
- Update routing
- Acknowledge/resolve alerts

### Tenant Admin

**Limited Access:**
- View own company data only
- View own performance metrics
- View own routing config

**Cannot:**
- View other companies
- Modify any configurations
- Access system-wide data

---

## Troubleshooting

### Common Issues

#### Cannot Login
- Verify credentials
- Check if account is active
- Clear browser cache
- Try different browser

#### Data Not Loading
- Check internet connection
- Refresh the page
- Check if backend is running
- Contact support if issue persists

#### Alerts Not Showing
- Check alert filters
- Verify alert configuration
- Check notification settings
- Ensure alerts are not all resolved

#### Server Showing Unhealthy
- Check server connectivity
- Verify server is running
- Check server logs
- Review health check configuration

### Performance Tips

1. **Use Filters**: Narrow down data to improve load times
2. **Pagination**: Use pagination for large datasets
3. **Cache**: Browser caches data for better performance
4. **Refresh**: Manual refresh updates real-time data

### Getting Help

**Support Channels:**
- Email: support@example.com
- Phone: +1-XXX-XXX-XXXX
- Documentation: docs.example.com
- Status Page: status.example.com

---

## Keyboard Shortcuts

- `Ctrl/Cmd + K`: Quick search
- `Ctrl/Cmd + /`: Show shortcuts
- `Esc`: Close modals
- `Arrow Keys`: Navigate lists

---

## Best Practices

1. **Regular Monitoring**: Check dashboard daily
2. **Alert Management**: Acknowledge alerts promptly
3. **Configuration Changes**: Test in staging first
4. **Documentation**: Document configuration changes
5. **Backup**: Keep backup of custom configurations

---

## Glossary

- **MPM**: Messages Per Minute
- **Z-Score**: Statistical measure of deviation
- **RLS**: Row Level Security
- **API**: Application Programming Interface
- **JWT**: JSON Web Token
- **RTL**: Right-to-Left (for Arabic)

---

## Appendix

### System Requirements

**Browser Support:**
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**Screen Resolution:**
- Minimum: 1280x720
- Recommended: 1920x1080

**Internet Connection:**
- Minimum: 5 Mbps
- Recommended: 10+ Mbps

---

**User Guide Version:** 1.0.0  
**Last Updated:** April 9, 2026  
**For Support:** support@example.com
