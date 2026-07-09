const WebSocket = require('ws');
const http = require('http');

/**
 * MockHub.js
 * 
 * A simple WebSocket server to simulate the ALHudhudAI Hub backend.
 * Used for end-to-end testing of the Mobile Gateway.
 */

const PORT = 8082;
const server = http.createServer((req, res) => {
    res.writeHead(200);
    res.end('ALHudhudAI Mock Hub is running\n');
});

const wss = new WebSocket.Server({ server, path: '/ws/pair' });

wss.on('connection', (ws, req) => {
    const url = new URL(req.url, `http://${req.headers.host}`);
    const deviceId = url.searchParams.get('device_id');
    const apiKey = url.searchParams.get('api_key');

    console.log(`\n[HUB] New connection from Device: ${deviceId}`);
    console.log(`[HUB] API Key: ${apiKey}`);

    ws.on('message', (message) => {
        const data = JSON.parse(message);
        console.log(`[HUB] Received from device:`, data);

        if (data.type === 'PING') {
            ws.send(JSON.stringify({ type: 'PONG' }));
        } else if (data.action === 'WAKE_UP') {
            console.log(`[HUB] Device is awake and ready.`);

            // --- Simulate sending a new message once paired ---
            setTimeout(() => {
                console.log(`[HUB] Sending test message to device...`);
                ws.send(JSON.stringify({
                    type: 'NEW_MESSAGE',
                    payload: {
                        id: 'TEST-' + Date.now(),
                        phone_number: '+966123456789',
                        content: 'هذه رسالة تجريبية من النظام للتأكد من نجاح الربط.',
                        idempotency_key: 'IDEM-' + Date.now()
                    }
                }));
            }, 3000);
        }
    });

    ws.on('close', () => {
        console.log(`[HUB] Device disconnected.`);
    });
});

server.listen(PORT, () => {
    console.log(`\n🚀 Mock Hub started on http://localhost:${PORT}`);
    console.log(`👉 In the app settings, set Server URL to: http://[YOUR_IP]:${PORT}`);
});
