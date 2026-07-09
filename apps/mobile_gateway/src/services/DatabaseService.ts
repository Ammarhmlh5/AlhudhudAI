import { open } from 'react-native-quick-sqlite';
import { runMigrations } from './migrations';

/**
 * DatabaseService.ts
 * 
 * Handles all SQLite operations for buffering messages before sending
 * and maintainig a local audit log.
 */

const db = open({ name: 'alhudhudai.db' });

// ✅ تفعيل Foreign Keys فوراً بعد فتح الاتصال
db.execute('PRAGMA foreign_keys = ON');

// ✅ تفعيل Write-Ahead Logging للأداء
db.execute('PRAGMA journal_mode = WAL');

// ✅ تحسين الأداء
db.execute('PRAGMA synchronous = NORMAL');
db.execute('PRAGMA temp_store = MEMORY');

export const initDatabase = () => {
    // التحقق من تفعيل Foreign Keys
    const fkCheck = db.execute('PRAGMA foreign_keys');
    const fkEnabled = fkCheck.rows?._array[0]?.foreign_keys === 1;
    
    console.log('✓ Foreign Keys enabled:', fkEnabled);
    
    if (!fkEnabled) {
        console.error('✗ Failed to enable Foreign Keys!');
    }
    
    db.execute(`
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      tenant_id INTEGER NOT NULL,
      phone_number TEXT NOT NULL,
      content TEXT NOT NULL,
      status TEXT CHECK(status IN ('PENDING', 'SENT', 'DELIVERED', 'FAILED', 'SKIPPED')) DEFAULT 'PENDING',
      failure_reason TEXT,
      attempts INTEGER DEFAULT 0,
      server_status_synced INTEGER DEFAULT 0,
      buffered_ack_sent INTEGER DEFAULT 0,
      idempotency_key TEXT UNIQUE,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      sent_at DATETIME,
      delivered_at DATETIME,
      UNIQUE(id, tenant_id)
    );
  `);

    try {
        db.execute(`
        CREATE TABLE IF NOT EXISTS logs (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          message_id TEXT,
          event TEXT NOT NULL,
          details TEXT,
          timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
        );
      `);
      // Create channels table for tracking device status
      db.execute(`
        CREATE TABLE IF NOT EXISTS channels (
          id TEXT PRIMARY KEY,
          status TEXT NOT NULL,
          last_seen DATETIME NOT NULL
        );
      `);
    } catch (e) {
        console.error('Error creating logs table:', e);
    }

    // Run migrations after all baseline tables are created
    try {
        runMigrations();
    } catch (e) {
        console.error('Error running migrations:', e);
    }

    // Create indexes safely after migrations have finalized the schema
    try {
        db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_tenant 
        ON messages(tenant_id);
      `);
    } catch (e: any) {
        console.log('Index idx_messages_tenant skipped (handled by migration):', e.message || e);
    }

    try {
        db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_status 
        ON messages(status);
      `);
    } catch (e: any) {
        console.log('Index idx_messages_status skipped:', e.message || e);
    }

    try {
        db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_tenant_status 
        ON messages(tenant_id, status);
      `);
    } catch (e: any) {
        console.log('Index idx_messages_tenant_status skipped:', e.message || e);
    }
};

export const DatabaseService = {
    // Message Operations
    saveMessage: (msg: { id: string, tenant_id: number, phone: string, content: string, idempotency_key?: string }) => {
        try {
            db.execute(
                'INSERT INTO messages (id, tenant_id, phone_number, content, idempotency_key) VALUES (?, ?, ?, ?, ?)',
                [msg.id, msg.tenant_id, msg.phone, msg.content, msg.idempotency_key || msg.id]
            );
            return true;
        } catch (e) {
            console.error('Failed to save message:', e);
            return false;
        }
    },

    getPendingMessages: () => {
        const results = db.execute(
            "SELECT * FROM messages WHERE status = 'PENDING' AND buffered_ack_sent = 1 ORDER BY created_at ASC LIMIT 10"
        );
        return results.rows?._array || [];
    },

    getAllPendingMessages: () => {
        try {
            const results = db.execute(
                "SELECT * FROM messages WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT 50"
            );
            return results.rows?._array || [];
        } catch (e) {
            console.error('Failed to get all pending messages:', e);
            return [];
        }
    },

    getMessageById: (id: string) => {
        const results = db.execute(
            'SELECT * FROM messages WHERE id = ? LIMIT 1',
            [id]
        );
        return results.rows?._array[0] || null;
    },

    updateMessageStatus: (id: string, status: string) => {
        try {
            const sentAt = status === 'SENT' ? new Date().toISOString() : null;
            db.execute(
                'UPDATE messages SET status = ?, sent_at = COALESCE(?, sent_at), server_status_synced = 0 WHERE id = ?',
                [status, sentAt, id]
            );
        } catch (e) {
            console.error('Failed to update message status:', e);
        }
    },

    markAsFailed: (id: string, reason?: string) => {
        db.execute(
            'UPDATE messages SET status = "FAILED", failure_reason = ?, server_status_synced = 0 WHERE id = ?',
            [reason || 'Unknown', id]
        );
    },

    incrementAttempts: (id: string) => {
        db.execute('UPDATE messages SET attempts = attempts + 1 WHERE id = ?', [id]);
    },

    markBufferedAckSent: (id: string) => {
        db.execute("UPDATE messages SET buffered_ack_sent = 1 WHERE id = ?", [id]);
    },

    // Log Operations

    // Channel status management
    updateChannelStatus: (id: string, status: string) => {
        const now = new Date().toISOString();
        // Try update first
        const result = db.execute('UPDATE channels SET status = ?, last_seen = ? WHERE id = ?', [status, now, id]);
        // If no rows affected, insert new record
        if (result?.rowsAffected === 0) {
            db.execute('INSERT INTO channels (id, status, last_seen) VALUES (?, ?, ?)', [id, status, now]);
        }
    },
    getChannelStatus: (id: string) => {
        const res = db.execute('SELECT status, last_seen FROM channels WHERE id = ?', [id]);
        return res?.rows?._array[0] || null;
    },

    // Log Operations
    addLog: (messageId: string | null, event: string, details: string) => {
        try {
            db.execute('INSERT INTO logs (message_id, event, details) VALUES (?, ?, ?)', [messageId, event, details]);
        } catch (e) {
            console.error('Failed to add log:', e);
        }
    },

    cleanupOldLogs: (days = 7) => {
        try {
            db.execute("DELETE FROM logs WHERE timestamp < datetime('now', '-' || ? || ' days')", [days]);
            db.execute("DELETE FROM messages WHERE status IN ('SENT', 'DELIVERED', 'FAILED') AND created_at < datetime('now', '-' || ? || ' days')", [days]);
            return true;
        } catch (e) {
            console.error('Cleanup failed:', e);
            return false;
        }
    },

    getStats: () => {
        const results = db.execute(`
            SELECT 
                (SELECT COUNT(*) FROM messages WHERE status = 'PENDING') as pending,
                (SELECT COUNT(*) FROM messages WHERE status = 'SENT') as sent,
                (SELECT COUNT(*) FROM messages WHERE status = 'FAILED') as failed
            FROM (SELECT 1)
        `);
        return results.rows?._array[0] || { pending: 0, sent: 0, failed: 0 };
    },

    getLogs: (limit = 50) => {
        const results = db.execute(
            "SELECT * FROM logs ORDER BY timestamp DESC LIMIT ?",
            [limit]
        );
        return results.rows?._array || [];
    },

    getHourlyStats: () => {
        const results = db.execute(`
            SELECT 
                strftime('%H', created_at) as hour,
                COUNT(*) as count
            FROM messages
            WHERE created_at > datetime('now', '-24 hours')
            GROUP BY hour
            ORDER BY hour ASC
        `);
        return results.rows?._array || [];
    },

    getFailedMessages: () => {
        const results = db.execute(
            "SELECT * FROM messages WHERE status = 'FAILED' ORDER BY created_at DESC"
        );
        return results.rows?._array || [];
    },

    getQueueMessages: () => {
        const results = db.execute(
            "SELECT * FROM messages WHERE status != 'FAILED' ORDER BY created_at DESC LIMIT 50"
        );
        return results.rows?._array || [];
    },

    getUnsyncedMessages: (): any[] => {
        const results = db.execute(
            "SELECT * FROM messages WHERE status IN ('SENT', 'DELIVERED', 'FAILED') AND server_status_synced = 0 LIMIT 50"
        );
        return results.rows?._array || [];
    },

    markAsSynced: (id: string) => {
        db.execute('UPDATE messages SET server_status_synced = 1 WHERE id = ?', [id]);
    },

    markAsUnsynced: (id: string) => {
        db.execute('UPDATE messages SET server_status_synced = 0 WHERE id = ?', [id]);
    },

    // ============================================================
    // Message Events (Event-Driven Buffer Table)
    // ============================================================

    saveEvent: (evt: { id: string, tenant_id: string, phone_number: string, content: string, sender_name?: string, message_type?: string, idempotency_key?: string }) => {
        try {
            db.execute(
                `INSERT OR IGNORE INTO message_events (id, tenant_id, phone_number, content, sender_name, message_type, idempotency_key) VALUES (?, ?, ?, ?, ?, ?, ?)`,
                [evt.id, evt.tenant_id, evt.phone_number, evt.content, evt.sender_name || '', evt.message_type || 'OUTBOUND_TEXT', evt.idempotency_key || evt.id]
            );
            return true;
        } catch (e) {
            console.error('Failed to save event:', e);
            return false;
        }
    },

    getNextPendingEvents: (limit = 5) => {
        try {
            const results = db.execute(
                `SELECT * FROM message_events WHERE status = 'BUFFERED' ORDER BY created_at ASC LIMIT ?`,
                [limit]
            );
            return results.rows?._array || [];
        } catch (e) {
            console.error('Failed to get pending events:', e);
            return [];
        }
    },

    markEventProcessing: (id: string) => {
        try {
            db.execute(
                `UPDATE message_events SET status = 'PROCESSING', attempts = attempts + 1 WHERE id = ? AND status = 'BUFFERED'`,
                [id]
            );
        } catch (e) {
            console.error('Failed to mark event processing:', e);
        }
    },

    markEventSent: (id: string) => {
        try {
            db.execute(
                `UPDATE message_events SET status = 'SENT', processed_at = ? WHERE id = ?`,
                [new Date().toISOString(), id]
            );
        } catch (e) {
            console.error('Failed to mark event sent:', e);
        }
    },

    markEventFailed: (id: string, reason?: string) => {
        try {
            db.execute(
                `UPDATE message_events SET status = 'FAILED', failure_reason = ?, processed_at = ? WHERE id = ?`,
                [reason || 'Unknown', new Date().toISOString(), id]
            );
        } catch (e) {
            console.error('Failed to mark event failed:', e);
        }
    },

    getEventStats: () => {
        try {
            const results = db.execute(`
                SELECT 
                    (SELECT COUNT(*) FROM message_events WHERE status = 'BUFFERED') as buffered,
                    (SELECT COUNT(*) FROM message_events WHERE status = 'PROCESSING') as processing,
                    (SELECT COUNT(*) FROM message_events WHERE status = 'SENT') as sent,
                    (SELECT COUNT(*) FROM message_events WHERE status = 'FAILED') as failed
                FROM (SELECT 1)
            `);
            return results.rows?._array[0] || { buffered: 0, processing: 0, sent: 0, failed: 0 };
        } catch (e) {
            console.error('Failed to get event stats:', e);
            return { buffered: 0, processing: 0, sent: 0, failed: 0 };
        }
    },

    // ============================================================
    // Status Reports Queue
    // ============================================================

    saveStatusReport: (messageId: string, status: string, errorCode?: string) => {
        try {
            db.execute(
                'INSERT INTO status_reports (message_id, status, error_code) VALUES (?, ?, ?)',
                [messageId, status, errorCode || null]
            );
            return true;
        } catch (e) {
            console.error('Failed to save status report:', e);
            return false;
        }
    },

    getPendingStatusReports: (limit = 10) => {
        try {
            const results = db.execute(
                `SELECT * FROM status_reports WHERE attempts < 5 ORDER BY attempts ASC, created_at ASC LIMIT ?`,
                [limit]
            );
            return results.rows?._array || [];
        } catch (e) {
            console.error('Failed to get pending status reports:', e);
            return [];
        }
    },

    incrementStatusReportAttempt: (id: number) => {
        try {
            db.execute(
                'UPDATE status_reports SET attempts = attempts + 1, last_attempt_at = ? WHERE id = ?',
                [new Date().toISOString(), id]
            );
        } catch (e) {
            console.error('Failed to increment status report attempt:', e);
        }
    },

    deleteStatusReport: (id: number) => {
        try {
            db.execute('DELETE FROM status_reports WHERE id = ?', [id]);
        } catch (e) {
            console.error('Failed to delete status report:', e);
        }
    },

    cleanupOldEvents: (days = 7) => {
        try {
            db.execute(
                "DELETE FROM message_events WHERE status IN ('SENT', 'FAILED') AND created_at < datetime('now', '-' || ? || ' days')",
                [days]
            );
            db.execute(
                "DELETE FROM status_reports WHERE created_at < datetime('now', '-' || ? || ' days')",
                [days]
            );
            return true;
        } catch (e) {
            console.error('Failed to cleanup old events:', e);
            return false;
        }
    }
};


// ============================================================
// Database Health Check Functions
// ============================================================

export const DatabaseHealth = {
    /**
     * التحقق من تفعيل Foreign Keys
     */
    checkForeignKeys: () => {
        const result = db.execute('PRAGMA foreign_keys');
        const enabled = result.rows?._array[0]?.foreign_keys === 1;
        return {
            enabled,
            status: enabled ? 'OK' : 'DISABLED',
            message: enabled ? 'Foreign Keys are enabled' : 'Foreign Keys are DISABLED!'
        };
    },
    
    /**
     * التحقق من سلامة قاعدة البيانات
     */
    checkIntegrity: () => {
        try {
            const result = db.execute('PRAGMA integrity_check');
            const status = result.rows?._array[0]?.integrity_check;
            return {
                ok: status === 'ok',
                status,
                message: status === 'ok' ? 'Database integrity OK' : 'Database has issues!'
            };
        } catch (e) {
            return {
                ok: false,
                status: 'ERROR',
                message: `Integrity check failed: ${e}`
            };
        }
    },
    
    /**
     * الحصول على معلومات قاعدة البيانات
     */
    getDatabaseInfo: () => {
        const pageCount = db.execute('PRAGMA page_count').rows?._array[0]?.page_count || 0;
        const pageSize = db.execute('PRAGMA page_size').rows?._array[0]?.page_size || 0;
        const freePages = db.execute('PRAGMA freelist_count').rows?._array[0]?.freelist_count || 0;
        const journalMode = db.execute('PRAGMA journal_mode').rows?._array[0]?.journal_mode;
        
        const sizeBytes = pageCount * pageSize;
        const sizeMB = (sizeBytes / (1024 * 1024)).toFixed(2);
        const freeBytes = freePages * pageSize;
        const freeMB = (freeBytes / (1024 * 1024)).toFixed(2);
        
        return {
            totalSize: `${sizeMB} MB`,
            freeSpace: `${freeMB} MB`,
            pageCount,
            pageSize,
            journalMode,
            foreignKeysEnabled: db.execute('PRAGMA foreign_keys').rows?._array[0]?.foreign_keys === 1
        };
    },
    
    /**
     * الحصول على قائمة الـ indexes
     */
    getIndexes: (tableName: string) => {
        const result = db.execute(`PRAGMA index_list(${tableName})`);
        return result.rows?._array || [];
    },
    
    /**
     * تحليل وتحسين قاعدة البيانات
     */
    optimize: () => {
        try {
            // تحليل الجداول لتحسين الاستعلامات
            db.execute('ANALYZE');
            
            // إعادة بناء قاعدة البيانات لتقليل الحجم
            db.execute('VACUUM');
            
            return {
                success: true,
                message: 'Database optimized successfully'
            };
        } catch (e) {
            return {
                success: false,
                message: `Optimization failed: ${e}`
            };
        }
    },
    
    /**
     * الحصول على إحصائيات شاملة
     */
    getFullStats: () => {
        const messageStats = db.execute(`
            SELECT 
                COUNT(*) as total,
                SUM(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END) as pending,
                SUM(CASE WHEN status = 'SENT' THEN 1 ELSE 0 END) as sent,
                SUM(CASE WHEN status = 'DELIVERED' THEN 1 ELSE 0 END) as delivered,
                SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed,
                AVG(attempts) as avg_attempts,
                MAX(attempts) as max_attempts
            FROM messages
        `).rows?._array[0];
        
        const logStats = db.execute(`
            SELECT COUNT(*) as total FROM logs
        `).rows?._array[0];
        
        const dbInfo = DatabaseHealth.getDatabaseInfo();
        const fkCheck = DatabaseHealth.checkForeignKeys();
        const integrityCheck = DatabaseHealth.checkIntegrity();
        
        return {
            messages: messageStats,
            logs: logStats,
            database: dbInfo,
            foreignKeys: fkCheck,
            integrity: integrityCheck
        };
    }
};
