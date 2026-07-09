/**
 * NotificationService.ts
 * 
 * Handles all notification operations on the mobile device.
 * Provides complete isolation between tenants (المستأجرين).
 */

import { open } from 'react-native-quick-sqlite';
import { v4 as uuidv4 } from 'uuid';

const db = open({ name: 'alhudhudai.db' });

export interface Notification {
    id: string;
    tenant_id: number;
    type: 'message' | 'status' | 'system' | 'delivery';
    title: string;
    body: string;
    data?: Record<string, any>;
    read: number;
    created_at: string;
    read_at?: string;
}

export interface CreateNotificationDTO {
    tenant_id: number;
    type: 'message' | 'status' | 'system' | 'delivery';
    title: string;
    body: string;
    data?: Record<string, any>;
}

class NotificationService {
    /**
     * Initialize the notifications table
     */
    init() {
        db.execute(`
            CREATE TABLE IF NOT EXISTS notifications (
                id TEXT PRIMARY KEY,
                tenant_id INTEGER NOT NULL,
                type TEXT NOT NULL CHECK(type IN ('message', 'status', 'system', 'delivery')),
                title TEXT NOT NULL,
                body TEXT NOT NULL,
                data TEXT,
                read INTEGER DEFAULT 0 CHECK(read IN (0, 1)),
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                read_at DATETIME,
                FOREIGN KEY (tenant_id) REFERENCES messages(tenant_id) ON DELETE CASCADE
            )
        `);

        // Create indexes for efficient queries
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_notifications_tenant 
            ON notifications(tenant_id)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_notifications_unread 
            ON notifications(tenant_id, read) 
            WHERE read = 0
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_notifications_created 
            ON notifications(tenant_id, created_at DESC)
        `);

        console.log('✅ Notifications table initialized');
    }

    /**
     * Create a new notification with tenant isolation
     * Each notification is strictly bound to its tenant
     */
    create(dto: CreateNotificationDTO): Notification {
        const id = uuidv4();
        const now = new Date().toISOString();
        
        db.execute(
            `INSERT INTO notifications (id, tenant_id, type, title, body, data, read, created_at) 
             VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
            [id, dto.tenant_id, dto.type, dto.title, dto.body, JSON.stringify(dto.data || {}), now]
        );

        return {
            id,
            tenant_id: dto.tenant_id,
            type: dto.type,
            title: dto.title,
            body: dto.body,
            data: dto.data,
            read: 0,
            created_at: now
        };
    }

    /**
     * Get notifications for a specific tenant only
     * ABSOLUTE TENANT ISOLATION - never returns notifications from other tenants
     */
    getByTenant(tenantId: number, limit = 50): Notification[] {
        const result = db.execute(
            `SELECT * FROM notifications 
             WHERE tenant_id = ? 
             ORDER BY created_at DESC 
             LIMIT ?`,
            [tenantId, limit]
        );

        return (result.rows?._array || []).map((row: any) => ({
            ...row,
            data: row.data ? JSON.parse(row.data) : undefined,
            read: Boolean(row.read)
        }));
    }

    /**
     * Get unread notifications count for a specific tenant
     * ABSOLUTE TENANT ISOLATION
     */
    getUnreadCount(tenantId: number): number {
        const result = db.execute(
            `SELECT COUNT(*) as count FROM notifications 
             WHERE tenant_id = ? AND read = 0`,
            [tenantId]
        );

        return result.rows?._array[0]?.count || 0;
    }

    /**
     * Get total unread count across all tenants (for badge display)
     * This is used when showing global notification count
     */
    getTotalUnreadCount(): number {
        const result = db.execute(
            `SELECT COUNT(*) as count FROM notifications WHERE read = 0`
        );

        return result.rows?._array[0]?.count || 0;
    }

    /**
     * Mark a notification as read
     * ABSOLUTE TENANT ISOLATION - only marks if notification belongs to tenant
     */
    markAsRead(notificationId: string, tenantId: number): boolean {
        const result = db.execute(
            `UPDATE notifications 
             SET read = 1, read_at = ? 
             WHERE id = ? AND tenant_id = ? AND read = 0`,
            [new Date().toISOString(), notificationId, tenantId]
        );

        return (result.rowsAffected || 0) > 0;
    }

    /**
     * Mark all notifications as read for a specific tenant
     * ABSOLUTE TENANT ISOLATION
     */
    markAllAsRead(tenantId: number): number {
        const result = db.execute(
            `UPDATE notifications 
             SET read = 1, read_at = ? 
             WHERE tenant_id = ? AND read = 0`,
            [new Date().toISOString(), tenantId]
        );

        return result.rowsAffected || 0;
    }

    /**
     * Delete a notification
     * ABSOLUTE TENANT ISOLATION - only deletes if notification belongs to tenant
     */
    delete(notificationId: string, tenantId: number): boolean {
        const result = db.execute(
            `DELETE FROM notifications WHERE id = ? AND tenant_id = ?`,
            [notificationId, tenantId]
        );

        return (result.rowsAffected || 0) > 0;
    }

    /**
     * Delete old notifications for a specific tenant
     * ABSOLUTE TENANT ISOLATION
     */
    deleteOldNotifications(tenantId: number, days = 30): number {
        const result = db.execute(
            `DELETE FROM notifications 
             WHERE tenant_id = ? AND created_at < datetime('now', '-' || ? || ' days')`,
            [tenantId, days]
        );

        return result.rowsAffected || 0;
    }

    /**
     * Get notification statistics for a specific tenant
     * ABSOLUTE TENANT ISOLATION
     */
    getStats(tenantId: number) {
        const result = db.execute(`
            SELECT 
                COUNT(*) as total,
                SUM(CASE WHEN read = 0 THEN 1 ELSE 0 END) as unread,
                SUM(CASE WHEN type = 'message' THEN 1 ELSE 0 END) as messages,
                SUM(CASE WHEN type = 'delivery' THEN 1 ELSE 0 END) as deliveries,
                SUM(CASE WHEN type = 'status' THEN 1 ELSE 0 END) as statuses,
                SUM(CASE WHEN type = 'system' THEN 1 ELSE 0 END) as systems
            FROM notifications 
            WHERE tenant_id = ?
        `, [tenantId]);

        return result.rows?._array[0] || { total: 0, unread: 0, messages: 0, deliveries: 0, statuses: 0, systems: 0 };
    }

    /**
     * Clear all notifications for a specific tenant
     * ABSOLUTE TENANT ISOLATION
     */
    clearAll(tenantId: number): number {
        const result = db.execute(
            `DELETE FROM notifications WHERE tenant_id = ?`,
            [tenantId]
        );

        return result.rowsAffected || 0;
    }

    /**
     * Create notification from WebSocket message
     * Automatically handles tenant isolation based on message data
     */
    createFromWebSocket(payload: {
        message_id?: string;
        tenant_id: number;
        type: 'message' | 'status' | 'system' | 'delivery';
        title: string;
        body: string;
        sender_name?: string;
        message_type?: string;
    }): Notification | null {
        // Validate tenant_id exists
        if (!payload.tenant_id || payload.tenant_id <= 0) {
            console.error('❌ Invalid tenant_id for notification');
            return null;
        }

        return this.create({
            tenant_id: payload.tenant_id,
            type: payload.type,
            title: payload.title,
            body: payload.body,
            data: {
                message_id: payload.message_id,
                sender_name: payload.sender_name,
                message_type: payload.message_type
            }
        });
    }
}

export const notificationService = new NotificationService();