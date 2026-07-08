import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { useAuth } from './AuthContext';
import { WS_BASE_URL } from '../config';

const SocketContext = createContext(null);

export const SocketProvider = ({ children }) => {
    const { user } = useAuth();
    const [socket, setSocket] = useState(null);
    const [incomingMessages, setIncomingMessages] = useState([]);
    const [deliveryUpdates, setDeliveryUpdates] = useState({});
    const [isConnected, setIsConnected] = useState(false);
    const [reconnectAttempt, setReconnectAttempt] = useState(0);
    const [metricsNonce, setMetricsNonce] = useState(0);
    const [messageUpdateNonce, setMessageUpdateNonce] = useState(0);
    const [heartbeatNonce, setHeartbeatNonce] = useState(0);
    const reconnectTimeoutRef = useRef(null);
    const pingIntervalRef = useRef(null);
    const socketRef = useRef(null);

    const scheduleReconnect = useCallback(() => {
        // Clear any existing reconnect timeout
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
            reconnectTimeoutRef.current = null;
        }

        // Don't reconnect if no user
        if (!user) {
            return;
        }

        // Calculate exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
        const baseDelay = 1000;
        const maxDelay = 30000;
        const delay = Math.min(baseDelay * Math.pow(2, reconnectAttempt), maxDelay);

        console.log(`Scheduling reconnect in ${delay}ms (attempt ${reconnectAttempt + 1})`);

        reconnectTimeoutRef.current = setTimeout(() => {
            setReconnectAttempt(prev => prev + 1);
            connect();
        }, delay);
    }, [reconnectAttempt, user]);

    const connect = useCallback(() => {
        if (!user) return;

        const token = localStorage.getItem('token');
        if (!token) {
            console.log('No token available, skipping WebSocket connection');
            return;
        }

        const wsUrl = `${WS_BASE_URL}/ws${token ? `?token=${token}` : ''}`;

        const ws = new WebSocket(wsUrl);
        socketRef.current = ws;

        ws.onopen = () => {
            // WebSocket Connected
            console.log('WebSocket connected');
            setIsConnected(true);
            setReconnectAttempt(0); // Reset reconnect attempts on successful connection
            
            // Start PING interval every 30 seconds
            if (pingIntervalRef.current) clearInterval(pingIntervalRef.current);
            pingIntervalRef.current = setInterval(() => {
                if (ws.readyState === WebSocket.OPEN) {
                    ws.send(JSON.stringify({ type: 'PING' }));
                }
            }, 30000);
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            handleSocketEvent(data);
        };

        ws.onclose = (event) => {
            // WebSocket Disconnected
            console.log('WebSocket disconnected', event.code, event.reason);
            setIsConnected(false);
            setSocket(null);
            socketRef.current = null;
            if (pingIntervalRef.current) {
                clearInterval(pingIntervalRef.current);
                pingIntervalRef.current = null;
            }

            // Attempt reconnection with exponential backoff
            scheduleReconnect();
        };

        ws.onerror = (err) => {
            // WebSocket Error handled by reconnect logic
            console.error('WebSocket error:', err);
            ws.close();
        };

        setSocket(ws);
    }, [user, scheduleReconnect]);

    const handleSocketEvent = (event) => {
        switch (event.type) {
            case 'MESSAGE_RECEIVED':
                setIncomingMessages(prev => [event.payload, ...prev].slice(0, 50));
                break;
            case 'MESSAGE_ANALYZED':
            case 'MESSAGE_SENT_TO_PHONE':
            case 'MESSAGE_STATUS_UPDATE':
            case 'MESSAGE_DISPATCHED':
            case 'DEVICE_STATUS_REPORT':
                setMessageUpdateNonce(n => n + 1);
                setDeliveryUpdates(prev => ({
                    ...prev,
                    [event.payload.incoming_id || event.payload.message_id]: event.payload
                }));
                break;
            case 'METRICS_UPDATED':
                setMetricsNonce(n => n + 1);
                break;
            case 'HEARTBEAT':
                setHeartbeatNonce(n => n + 1);
                break;
            default:
                break;
        }
    };

    useEffect(() => {
        connect();
        return () => {
            // Clear reconnect timeout on unmount
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
                reconnectTimeoutRef.current = null;
            }
            if (pingIntervalRef.current) {
                clearInterval(pingIntervalRef.current);
                pingIntervalRef.current = null;
            }
            // Close socket on unmount
            if (socketRef.current) {
                socketRef.current.close();
            }
        };
    }, [user]);

    return (
        <SocketContext.Provider value={{ 
            isConnected, 
            incomingMessages, 
            deliveryUpdates,
            metricsNonce,
            messageUpdateNonce,
            heartbeatNonce,
            reconnectAttempt
        }}>
            {children}
        </SocketContext.Provider>
    );
};

export const useSocket = () => {
    const context = useContext(SocketContext);
    if (!context) {
        throw new Error('useSocket must be used within a SocketProvider');
    }
    return context;
};
