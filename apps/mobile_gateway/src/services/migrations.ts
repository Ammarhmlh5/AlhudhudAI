import { open } from 'react-native-quick-sqlite';

/**
 * migrations.ts
 * 
 * Database migration system for Mobile Gateway
 * Handles schema upgrades for existing databases
 */

const db = open({ name: 'alhudhudai.db' });

interface Migration {
  version: number;
  name: string;
  up: () => void;
}

const migrations: Migration[] = [
  {
    version: 1,
    name: 'add_tenant_id_and_indexes',
    up: () => {
      // Check if messages table exists first (fresh installs will not have it yet)
      const tableExists = db.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='messages'");
      const hasTable = (tableExists.rows?._array || []).length > 0;
      if (!hasTable) {
        console.log('Table messages does not exist yet. Skipping migration 1 since the table will be created with the latest schema.');
        return;
      }

      // Check if tenant_id column exists
      const tableInfo = db.execute('PRAGMA table_info(messages)');
      const columns = tableInfo.rows?._array || [];
      const hasTenantId = columns.some((col: any) => col.name === 'tenant_id');

      if (!hasTenantId) {
        // Add tenant_id column if it doesn't exist
        db.execute('ALTER TABLE messages ADD COLUMN tenant_id INTEGER NOT NULL DEFAULT 0');
        console.log('Added tenant_id column to messages table');
      }

      // Create indexes if they don't exist
      db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_tenant 
        ON messages(tenant_id)
      `);

      db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_status 
        ON messages(status)
      `);

      db.execute(`
        CREATE INDEX IF NOT EXISTS idx_messages_tenant_status 
        ON messages(tenant_id, status)
      `);

      console.log('Created performance indexes');

      // Add UNIQUE constraint on (id, tenant_id) if not exists
      // Note: SQLite doesn't support adding constraints to existing tables
      // We need to recreate the table if the constraint doesn't exist
      const indexes = db.execute('PRAGMA index_list(messages)');
      const hasUniqueConstraint = (indexes.rows?._array || []).some(
        (idx: any) => idx.name.includes('unique') && idx.unique === 1
      );

      if (!hasUniqueConstraint) {
        console.log('Recreating table with UNIQUE constraint...');
        
        // Create new table with correct schema
        db.execute(`
          CREATE TABLE IF NOT EXISTS messages_new (
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
          )
        `);

        // Copy data from old table
        db.execute(`
          INSERT INTO messages_new 
          SELECT * FROM messages
        `);

        // Drop old table
        db.execute('DROP TABLE messages');

        // Rename new table
        db.execute('ALTER TABLE messages_new RENAME TO messages');

        // Recreate indexes
        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_messages_tenant 
          ON messages(tenant_id)
        `);

        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_messages_status 
          ON messages(status)
        `);

        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_messages_tenant_status 
          ON messages(tenant_id, status)
        `);

        console.log('Table recreated with UNIQUE constraint');
      }
    },
  },
  {
    version: 2,
    name: 'add_foreign_keys_indexes_and_constraints',
    up: () => {
        console.log('Migration 2: Adding Foreign Keys, Indexes, and Constraints...');
        
        // ============================================================
        // 1. إضافة Indexes على جدول logs
        // ============================================================
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_message_id 
            ON logs(message_id)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_timestamp 
            ON logs(timestamp DESC)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_event 
            ON logs(event)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_message_timestamp 
            ON logs(message_id, timestamp DESC)
        `);
        
        console.log('✓ Logs indexes created');
        
        // ============================================================
        // 2. إضافة Indexes إضافية على جدول messages
        // ============================================================
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_messages_created_at 
            ON messages(created_at DESC)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_messages_idempotency 
            ON messages(idempotency_key)
        `);
        
        console.log('✓ Additional messages indexes created');
        
        // ============================================================
        // 3. التحقق من النتائج
        // ============================================================
        const messagesIndexes = db.execute('PRAGMA index_list(messages)');
        const logsIndexes = db.execute('PRAGMA index_list(logs)');
        
        console.log('Messages indexes count:', messagesIndexes.rows?._array.length);
        console.log('Logs indexes count:', logsIndexes.rows?._array.length);
        
        console.log('✓ Migration 2 completed successfully');
    },
  },
  {
    version: 3,
    name: 'create_message_events_table',
    up: () => {
        console.log('Migration 3: Creating message_events table for event-driven processing...');
        
        // Create message_events table for buffering incoming messages
        // This table acts as a reliable event buffer - messages are saved here
        // immediately on arrival and then processed gradually by SmsQueueManager
        db.execute(`
          CREATE TABLE IF NOT EXISTS message_events (
            id TEXT PRIMARY KEY,
            tenant_id TEXT NOT NULL,
            phone_number TEXT NOT NULL,
            content TEXT NOT NULL,
            sender_name TEXT DEFAULT '',
            message_type TEXT DEFAULT 'OUTBOUND_TEXT',
            status TEXT CHECK(status IN ('BUFFERED', 'PROCESSING', 'SENT', 'FAILED')) DEFAULT 'BUFFERED',
            failure_reason TEXT,
            attempts INTEGER DEFAULT 0,
            idempotency_key TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            processed_at DATETIME,
            UNIQUE(id, tenant_id)
          )
        `);
        
        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_message_events_status 
          ON message_events(status)
        `);
        
        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_message_events_created 
          ON message_events(created_at ASC)
        `);
        
        console.log('✓ message_events table created');
    },
  },
  {
    version: 4,
    name: 'create_status_reports_table',
    up: () => {
        console.log('Migration 4: Creating status_reports table for reliable status reporting...');
        
        // Create status_reports table for queuing status reports
        // Status reports are saved here when WebSocket is unavailable
        // and sent when connection is restored
        db.execute(`
          CREATE TABLE IF NOT EXISTS status_reports (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            message_id TEXT NOT NULL,
            status TEXT NOT NULL,
            error_code TEXT,
            attempts INTEGER DEFAULT 0,
            last_attempt_at DATETIME,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
          )
        `);
        
        db.execute(`
          CREATE INDEX IF NOT EXISTS idx_status_reports_unsent 
          ON status_reports(attempts, created_at ASC)
        `);
        
        console.log('✓ status_reports table created');
    },
  },
];

export const runMigrations = () => {
  // Create migrations table if it doesn't exist
  db.execute(`
    CREATE TABLE IF NOT EXISTS schema_migrations (
      version INTEGER PRIMARY KEY,
      name TEXT NOT NULL,
      applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Get current version
  const result = db.execute('SELECT MAX(version) as version FROM schema_migrations');
  const currentVersion = result.rows?._array[0]?.version || 0;

  console.log(`Current database version: ${currentVersion}`);

  // Run pending migrations
  const pendingMigrations = migrations.filter(m => m.version > currentVersion);

  if (pendingMigrations.length === 0) {
    console.log('Database is up to date');
    return;
  }

  console.log(`Running ${pendingMigrations.length} pending migrations...`);

  pendingMigrations.forEach(migration => {
    try {
      console.log(`Applying migration ${migration.version}: ${migration.name}`);
      migration.up();
      
      // Record migration
      db.execute(
        'INSERT INTO schema_migrations (version, name) VALUES (?, ?)',
        [migration.version, migration.name]
      );
      
      console.log(`Migration ${migration.version} completed successfully`);
    } catch (error) {
      console.error(`Migration ${migration.version} failed:`, error);
      throw error;
    }
  });

  console.log('All migrations completed successfully');
};
