import { useAppStore } from './StorageService';
import { DatabaseService } from './DatabaseService';
import NetInfo, { NetInfoSubscription } from '@react-native-community/netinfo';

/**
 * WebSocketService.ts
 * 
 * Manages the persistent connection to ALHudhudAI Hub.
 * Handles incoming message requests and status acknowledgments.
 */

class WebSocketService {
    private socket: WebSocket | null = null;
    private reconnectTimer: any = null;
    private heartbeatTimer: any = null;
    private reconnectAttempts: number = 0;
    private netInfoUnsubscribe: NetInfoSubscription | null = null;

    public connect() {
        const { apiUrl, apiKey, deviceUuid, isBanned } = useAppStore.getState();

        if (isBanned) {
            console.error('Device is banned. WebSocket connection aborted.');
            return;
        }

        if (!apiUrl || !apiKey) {
            console.warn('API credentials missing. Retrying in 30s...');
            this.scheduleReconnectWithDelay(30000); 
            return;
        }

        // Initialize NetInfo listener for instantaneous network recovery
        if (!this.netInfoUnsubscribe) {
            this.netInfoUnsubscribe = NetInfo.addEventListener(state => {
                if (state.isConnected && this.socket?.readyState !== WebSocket.OPEN) {
                    console.log('Network connected. Triggering instantaneous WebSocket reconnection...');
                    this.connect();
                }
            });
        }

        const wsUrl = apiUrl.replace(/^http/, 'ws');
        console.log(`Connecting to WebSocket: ${wsUrl}/ws/pair`);

        this.socket = new WebSocket(`${wsUrl}/ws/pair?api_key=${apiKey}&device_id=${deviceUuid}`);

        this.socket.onopen = () => {
            // Register channel status as ONLINE in local DB
            if (deviceUuid) DatabaseService.updateChannelStatus(deviceUuid, 'ONLINE');
            this.reconnectAttempts = 0;
            useAppStore.getState().setConnected(true);
            DatabaseService.addLog(null, 'WS_CONNECTED', 'Secure WebSocket channel established.');
            this.sendWakeUp();
            this.startHeartbeat();
            this.syncPendingStatuses(); // Sync missed reports on connect
        };

        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'PONG') {
                    return;
                }
                if (data.type === 'NEW_MESSAGE' || data.type === 'MESSAGE_DELIVERY' || data.type === 'MESSAGE') {
                    const payload = data.payload || data;
                    this.handleIncomingMessage({
                        id: payload.message_id || data.message_id,
                        tenant_id: payload.tenant_id || data.tenant_id,
                        phone_number: payload.phone_number || data.phone_number,
                        content: payload.content || data.content,
                        sender_name: payload.sender_name || data.sender_name,
                        message_type: payload.message_type || data.message_type,
                        idempotency_key: payload.idempotency_key || data.idempotency_key || ''
                    });
                }
            } catch (e) { 
                console.error('Error parsing WebSocket message:', e);
            }
        };

        this.socket.onclose = (e) => {
            console.log(`WebSocket closed: ${e.reason}. Retrying in 5s...`);
            useAppStore.getState().setConnected(false);
            this.stopHeartbeat();
            this.scheduleReconnect();
        };

        this.socket.onerror = (e) => {
            console.error('WebSocket Error:', e);
            useAppStore.getState().setConnected(false);
        };
    }

    private startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatTimer = setInterval(() => {
            if (this.socket?.readyState === WebSocket.OPEN) {
                this.socket.send(JSON.stringify({ type: 'PING' }));
                this.syncPendingStatuses(); // Try to sync any unsent reports
            } else {
                this.connect(); // Force reconnect if stuck
            }
        }, 30000); // 30 seconds (matches backend heartbeat interval)
    }

    public disconnect() {
        const { deviceUuid } = useAppStore.getState();
        // Register channel status as OFFLINE in local DB
        if (deviceUuid) {
            DatabaseService.updateChannelStatus(deviceUuid, 'OFFLINE');
        }
        
        // Existing disconnect logic
        this.stopHeartbeat();
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.netInfoUnsubscribe) {
            this.netInfoUnsubscribe();
            this.netInfoUnsubscribe = null;
        }
        if (this.socket) {
            this.socket.close(1000, "Service stopped gracefully");
            this.socket = null;
        }
        console.log('WebSocket connection cleanly disconnected.');
        useAppStore.getState().setConnected(false);
    }

    private syncPendingStatuses() {
        if (this.socket?.readyState !== WebSocket.OPEN) return;

        // Flush legacy unsynced messages
        const unsynced = DatabaseService.getUnsyncedMessages();
        if (unsynced.length > 0) {
            console.log(`[SYNC] Found ${unsynced.length} unsynced status reports. Processing...`);
            unsynced.forEach(msg => {
                const success = this.sendStatusReport(msg.id, msg.status, msg.failure_reason);
                if (success) {
                    DatabaseService.markAsSynced(msg.id);
                }
            });
        }

        // Flush new status reports queue
        this.flushStatusReports();
    }

    private stopHeartbeat() {
        if (this.heartbeatTimer) clearInterval(this.heartbeatTimer);
    }

    private async handleIncomingMessage(payload: any) {
        const startTime = Date.now();

        // Log message receipt
        console.log(`[WS] Message received: ${payload.id} from ${payload.phone_number}`);

        // Step 1: Save to message_events table immediately (BEFORE ACK)
        // This is the critical event buffer - saves first, processes later
        const success = DatabaseService.saveEvent({
            id: payload.id,
            tenant_id: payload.tenant_id || payload.tenant_id?.toString(),
            phone_number: payload.phone_number,
            content: payload.content,
            sender_name: payload.sender_name,
            message_type: payload.message_type,
            idempotency_key: payload.idempotency_key
        });

        if (!success) {
            console.error(`[WS] Failed to buffer event ${payload.id}`);
            return;
        }

        console.log(`[WS] Event ${payload.id} saved to message_events`);
        DatabaseService.addLog(payload.id, 'EVENT_BUFFERED', 'Message saved to event buffer.');

        // Step 2: Send BUFFERED status report (queued for reliability)
        const reportQueued = DatabaseService.saveStatusReport(payload.id, 'BUFFERED');
        if (reportQueued) {
            // Try sending immediately, will be retried from queue if fails
            this.flushStatusReports();
            const elapsed = Date.now() - startTime;
            console.log(`[WS] BUFFERED report queued in ${elapsed}ms`);
            DatabaseService.addLog(payload.id, 'BUFFERED_QUEUED', `BUFFERED report queued in ${elapsed}ms`);
        }
    }

    public flushStatusReports() {
        if (this.socket?.readyState !== WebSocket.OPEN) return;

        const pending = DatabaseService.getPendingStatusReports(10);
        if (pending.length === 0) return;

        for (const report of pending) {
            const sent = this.sendStatusReport(report.message_id, report.status, report.error_code);
            if (sent) {
                DatabaseService.deleteStatusReport(report.id);
            } else {
                DatabaseService.incrementStatusReportAttempt(report.id);
            }
        }
    }

    private sendWakeUp() {
        if (this.socket?.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({ action: 'WAKE_UP' }));
        }
    }

    private scheduleReconnect() {
        if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
        
        // Exponential backoff: 5s, 10s, 20s, 40s... cap at 60s
        const delay = Math.min(5000 * Math.pow(2, this.reconnectAttempts), 60000);
        this.reconnectAttempts++;
        
        console.log(`WebSocket reconnect scheduled in ${delay}ms (Attempt ${this.reconnectAttempts})`);
        this.reconnectTimer = setTimeout(() => this.connect(), delay);
    }

    private scheduleReconnectWithDelay(delay: number) {
        if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
        this.reconnectTimer = setTimeout(() => this.connect(), delay);
    }

    public sendStatusReport(messageId: string, status: string, errorCode?: string): boolean {
        if (this.socket?.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({
                type: 'STATUS_REPORT',
                message_id: messageId,
                status: status,
                error_code: errorCode, // Detailed failure reasons (Report 2)
                timestamp: new Date().toISOString()
            }));
            return true;
        }
        return false;
    }
}

export const wsService = new WebSocketService();
