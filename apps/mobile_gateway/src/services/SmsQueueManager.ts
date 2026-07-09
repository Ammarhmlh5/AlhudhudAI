import { DatabaseService } from './DatabaseService';
import { wsService } from './WebSocketService';
import { NativeSms } from './NativeSms';

/**
 * SmsQueueManager.ts
 *
 * SMS Processing with Event Buffer (Gradual Processing)
 * - Polls message_events table every 1-2 seconds
 * - Processes up to 5 events per cycle
 * - Marks events BUFFERED → PROCESSING → SENT/FAILED
 * - Sends status reports via status_reports queue
 * - Moves completed events to messages table for history
 */

class SmsQueueManager {
    private pollTimer: any = null;
    private processing = false;
    private batchSize = 5;
    private pollInterval = 1500;
    private retryDelays = [1000, 3000, 9000];

    public start() {
        if (this.pollTimer) return;
        console.log(`SMS Queue Manager started (Polling every ${this.pollInterval}ms)`);
        this.processNextBatch();
        this.pollTimer = setInterval(() => this.processNextBatch(), this.pollInterval);
    }

    public stop() {
        if (this.pollTimer) {
            clearInterval(this.pollTimer);
            this.pollTimer = null;
        }
        this.processing = false;
        console.log('SMS Queue Manager stopped');
    }

    private async processNextBatch() {
        if (this.processing) return;
        this.processing = true;

        try {
            const events = DatabaseService.getNextPendingEvents(this.batchSize);
            if (events.length === 0) {
                return;
            }

            console.log(`[QUEUE] Processing ${events.length} events from buffer`);

            for (const event of events) {
                DatabaseService.markEventProcessing(event.id);
                DatabaseService.addLog(event.id, 'EVENT_PROCESSING', 'Processing event from buffer');

                await this.sendEventWithRetry(event);

                await this.sleep(200);
            }
        } finally {
            this.processing = false;
        }
    }

    private async sendEventWithRetry(event: any) {
        const maxAttempts = 3;
        let lastError = '';

        for (let attempt = 0; attempt < maxAttempts; attempt++) {
            const attemptNum = attempt + 1;
            console.log(`[QUEUE] Sending ${event.id} (Attempt ${attemptNum}/${maxAttempts})`);

            const startTime = Date.now();

            try {
                const success = await NativeSms.send(
                    event.id,
                    event.phone_number,
                    event.content
                );

                const elapsed = Date.now() - startTime;

                if (success) {
                    DatabaseService.markEventSent(event.id);
                    DatabaseService.saveMessage({
                        id: event.id,
                        tenant_id: parseInt(event.tenant_id) || 0,
                        phone: event.phone_number,
                        content: event.content,
                        idempotency_key: event.idempotency_key || event.id
                    });
                    DatabaseService.updateMessageStatus(event.id, 'SENT');
                    DatabaseService.addLog(
                        event.id,
                        'MSG_SENT',
                        `Sent successfully in ${elapsed}ms (Attempt ${attemptNum})`
                    );

                    DatabaseService.saveStatusReport(event.id, 'SENT');
                    wsService.flushStatusReports?.();

                    return;
                } else {
                    lastError = 'Native SMS send returned false';
                    DatabaseService.addLog(event.id, 'SEND_FAILED', `Attempt ${attemptNum}: ${lastError}`);
                }
            } catch (error: any) {
                lastError = error.message || 'Unknown error';
                DatabaseService.addLog(event.id, 'SEND_ERROR', `Attempt ${attemptNum}: ${lastError}`);
            }

            if (attempt < maxAttempts - 1) {
                await this.sleep(this.retryDelays[attempt]);
            }
        }

        DatabaseService.markEventFailed(event.id, lastError);
        DatabaseService.addLog(event.id, 'MSG_FAILED', `All ${maxAttempts} attempts failed: ${lastError}`);
        DatabaseService.saveStatusReport(event.id, 'FAILED', lastError);
        wsService.flushStatusReports?.();
    }

    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    public getStatus() {
        const stats = DatabaseService.getEventStats();
        return {
            processing: this.processing,
            ...stats,
        };
    }
}

export const smsQueue = new SmsQueueManager();
