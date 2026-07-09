/**
 * NotificationService.test.ts
 * 
 * Unit tests for the NotificationService with tenant isolation.
 * Tests cover all notification operations including CRUD, tenant isolation,
 * and WebSocket integration.
 */

import { notificationService, Notification, CreateNotificationDTO } from '../NotificationService';

// Mock react-native-quick-sqlite
jest.mock('react-native-quick-sqlite', () => ({
    open: jest.fn(() => ({
        execute: jest.fn(() => ({ rowsAffected: 1, rows: { _array: [] } })),
    })),
}));

// Mock uuid
jest.mock('uuid', () => ({
    v4: jest.fn(() => 'test-uuid-1234'),
}));

describe('NotificationService', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    describe('init()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should create notifications table with correct schema', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            // Re-import to get fresh instance with mock
            const { notificationService: freshService } = require('../NotificationService');
            freshService.init();

            expect(mockDb.execute).toHaveBeenCalled();
            
            // Verify table creation
            const createTableCall = mockDb.execute.mock.calls[0][0];
            expect(createTableCall).toContain('CREATE TABLE IF NOT EXISTS notifications');
            expect(createTableCall).toContain('tenant_id');
            expect(createTableCall).toContain('type');
            expect(createTableCall).toContain('read');
        });

        it('should create indexes for efficient queries', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            freshService.init();

            // Check for index creation calls (at least 3 indexes)
            expect(mockDb.execute.mock.calls.length).toBeGreaterThanOrEqual(4);
        });
    });

    describe('create()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should create a notification with tenant isolation', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const dto: CreateNotificationDTO = {
                tenant_id: 1,
                type: 'message',
                title: 'Test Notification',
                body: 'Test body',
                data: { key: 'value' },
            };

            const result = freshService.create(dto);

            expect(result.id).toBe('test-uuid-1234');
            expect(result.tenant_id).toBe(1);
            expect(result.type).toBe('message');
            expect(result.title).toBe('Test Notification');
            expect(result.read).toBe(0);
            expect(mockDb.execute).toHaveBeenCalled();
        });

        it('should set read to 0 by default', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const dto: CreateNotificationDTO = {
                tenant_id: 1,
                type: 'system',
                title: 'System Alert',
                body: 'System message',
            };

            const result = freshService.create(dto);

            expect(result.read).toBe(0);
            expect(result.read_at).toBeUndefined();
        });

        it('should serialize data to JSON string', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const dto: CreateNotificationDTO = {
                tenant_id: 1,
                type: 'delivery',
                title: 'Delivery Update',
                body: 'Message delivered',
                data: { message_id: 'msg-123', status: 'delivered' },
            };

            freshService.create(dto);

            const executeCall = mockDb.execute.mock.calls[0];
            const dataParam = executeCall[1][5];
            expect(dataParam).toBe('{"message_id":"msg-123","status":"delivered"}');
        });
    });

    describe('getByTenant()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should return notifications for specific tenant only', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: {
                        _array: [
                            { id: '1', tenant_id: 1, type: 'message', title: 'Test', body: 'Body', data: '{}', read: 0 },
                        ],
                    },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getByTenant(1);

            expect(result.length).toBe(1);
            expect(result[0].tenant_id).toBe(1);
            
            // Verify tenant_id is used in query
            const queryCall = mockDb.execute.mock.calls[0];
            expect(queryCall[1]).toContain(1);
        });

        it('should parse JSON data field', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: {
                        _array: [
                            { id: '1', tenant_id: 1, type: 'message', title: 'Test', body: 'Body', data: '{"key":"value"}', read: 0 },
                        ],
                    },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getByTenant(1);

            expect(result[0].data).toEqual({ key: 'value' });
        });

        it('should convert read integer to boolean', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: {
                        _array: [
                            { id: '1', tenant_id: 1, type: 'message', title: 'Test', body: 'Body', data: '{}', read: 1 },
                        ],
                    },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getByTenant(1);

            expect(result[0].read).toBe(true);
        });

        it('should return empty array when no notifications', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getByTenant(999);

            expect(result).toEqual([]);
        });

        it('should respect limit parameter', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            freshService.getByTenant(1, 25);

            const queryCall = mockDb.execute.mock.calls[0];
            expect(queryCall[1]).toContain(25);
        });
    });

    describe('getUnreadCount()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should return unread count for specific tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [{ count: 5 }] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getUnreadCount(1);

            expect(result).toBe(5);
        });

        it('should return 0 when no unread notifications', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [{ count: 0 }] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getUnreadCount(1);

            expect(result).toBe(0);
        });
    });

    describe('getTotalUnreadCount()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should return total unread count across all tenants', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [{ count: 10 }] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getTotalUnreadCount();

            expect(result).toBe(10);
        });
    });

    describe('markAsRead()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should mark notification as read with tenant isolation', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.markAsRead('notif-123', 1);

            expect(result).toBe(true);
            
            const queryCall = mockDb.execute.mock.calls[0];
            expect(queryCall[0]).toContain('UPDATE notifications');
            expect(queryCall[0]).toContain('tenant_id');
        });

        it('should return false when notification not found', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.markAsRead('not-found', 1);

            expect(result).toBe(false);
        });

        it('should only mark if notification belongs to tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            // Try to mark notification with wrong tenant
            const result = freshService.markAsRead('notif-123', 999);

            expect(result).toBe(false);
            
            const queryCall = mockDb.execute.mock.calls[0];
            expect(queryCall[1]).toContain('notif-123');
            expect(queryCall[1]).toContain(999);
        });
    });

    describe('markAllAsRead()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should mark all notifications as read for tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 5 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.markAllAsRead(1);

            expect(result).toBe(5);
        });
    });

    describe('delete()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should delete notification with tenant isolation', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.delete('notif-123', 1);

            expect(result).toBe(true);
        });

        it('should return false when notification not found', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.delete('not-found', 1);

            expect(result).toBe(false);
        });
    });

    describe('deleteOldNotifications()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should delete old notifications for specific tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 10 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.deleteOldNotifications(1, 30);

            expect(result).toBe(10);
        });

        it('should use default 30 days if not specified', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 5 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            freshService.deleteOldNotifications(1);

            const queryCall = mockDb.execute.mock.calls[0];
            expect(queryCall[1]).toContain(30);
        });
    });

    describe('getStats()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should return notification statistics for tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: {
                        _array: [{
                            total: 100,
                            unread: 25,
                            messages: 50,
                            deliveries: 30,
                            statuses: 10,
                            systems: 10,
                        }],
                    },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getStats(1);

            expect(result.total).toBe(100);
            expect(result.unread).toBe(25);
            expect(result.messages).toBe(50);
            expect(result.deliveries).toBe(30);
        });

        it('should return zeros when no notifications', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({
                    rows: { _array: [] },
                }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.getStats(1);

            expect(result.total).toBe(0);
            expect(result.unread).toBe(0);
        });
    });

    describe('clearAll()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should clear all notifications for specific tenant', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 50 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');
            const result = freshService.clearAll(1);

            expect(result).toBe(50);
        });
    });

    describe('createFromWebSocket()', () => {
        beforeEach(() => {
            jest.resetModules();
            jest.clearAllMocks();
        });

        it('should create notification from WebSocket payload', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const payload = {
                tenant_id: 1,
                type: 'message' as const,
                title: 'New Message',
                body: 'You have a new message',
                message_id: 'msg-123',
                sender_name: 'John Doe',
                message_type: 'text',
            };

            const result = freshService.createFromWebSocket(payload);

            expect(result).not.toBeNull();
            expect(result?.tenant_id).toBe(1);
            expect(result?.type).toBe('message');
        });

        it('should return null for invalid tenant_id', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const payload = {
                tenant_id: 0,
                type: 'message' as const,
                title: 'Test',
                body: 'Test body',
            };

            const result = freshService.createFromWebSocket(payload);

            expect(result).toBeNull();
        });

        it('should return null for missing tenant_id', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 1 }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            const { notificationService: freshService } = require('../NotificationService');

            const payload = {
                type: 'message' as const,
                title: 'Test',
                body: 'Test body',
            } as any;

            const result = freshService.createFromWebSocket(payload);

            expect(result).toBeNull();
        });
    });

    describe('Tenant Isolation', () => {
        it('should enforce tenant isolation in all queries', () => {
            const { open } = require('react-native-quick-sqlite');
            const mockDb = {
                execute: jest.fn().mockReturnValue({ rowsAffected: 0, rows: { _array: [] } }),
            };
            (open as jest.Mock).mockReturnValue(mockDb);

            // Test getByTenant
            notificationService.getByTenant(1);
            let query = mockDb.execute.mock.calls[0][0];
            expect(query).toContain('tenant_id');

            // Test getUnreadCount
            notificationService.getUnreadCount(1);
            query = mockDb.execute.mock.calls[1][0];
            expect(query).toContain('tenant_id');

            // Test markAsRead
            notificationService.markAsRead('id', 1);
            query = mockDb.execute.mock.calls[2][0];
            expect(query).toContain('tenant_id');

            // Test delete
            notificationService.delete('id', 1);
            query = mockDb.execute.mock.calls[3][0];
            expect(query).toContain('tenant_id');

            // Test clearAll
            notificationService.clearAll(1);
            query = mockDb.execute.mock.calls[4][0];
            expect(query).toContain('tenant_id');

            // Test deleteOldNotifications
            notificationService.deleteOldNotifications(1);
            query = mockDb.execute.mock.calls[5][0];
            expect(query).toContain('tenant_id');

            // Test getStats
            notificationService.getStats(1);
            query = mockDb.execute.mock.calls[6][0];
            expect(query).toContain('tenant_id');
        });
    });
});