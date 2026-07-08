#!/usr/bin/env python3
"""
Apply PostgreSQL Migration Script
"""

import sys
import psycopg2
from psycopg2 import sql

def apply_migration(migration_file):
    """Apply a migration file to the database"""
    
    # Database connection parameters
    conn_params = {
        'host': 'localhost',
        'port': 5433,
        'user': 'postgres',
        'password': 'Ammar1983hmlh##',
        'database': 'al_hudhud_new'
    }
    
    try:
        # Connect to database
        print("Connecting to database...")
        conn = psycopg2.connect(**conn_params)
        conn.autocommit = False
        cursor = conn.cursor()
        
        print("✓ Connected successfully")
        
        # Read migration file
        print(f"Reading migration file: {migration_file}")
        with open(migration_file, 'r', encoding='utf-8') as f:
            migration_sql = f.read()
        
        # Execute migration
        print("Applying migration...")
        cursor.execute(migration_sql)
        
        # Commit transaction
        conn.commit()
        print("✓ Migration applied successfully")
        
        # Close connection
        cursor.close()
        conn.close()
        
        return True
        
    except psycopg2.Error as e:
        print(f"✗ Database error: {e}")
        if conn:
            conn.rollback()
        return False
    except FileNotFoundError:
        print(f"✗ Migration file not found: {migration_file}")
        return False
    except Exception as e:
        print(f"✗ Error: {e}")
        return False

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python apply_migration.py <migration_file>")
        sys.exit(1)
    
    migration_file = sys.argv[1]
    success = apply_migration(migration_file)
    
    sys.exit(0 if success else 1)
