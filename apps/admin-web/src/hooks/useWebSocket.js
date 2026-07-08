import { useEffect, useRef, useState, useCallback } from 'react';

/**
 * useWebSocket Hook
 * 
 * Manages WebSocket connection with auto-reconnect functionality
 * Provides real-time updates for server health monitoring
 * 
 * @param {string} url - WebSocket URL
 * @param {Object} options - Configuration options
 * @param {boolean} options.enabled - Enable/disable connection
 * @param {number} options.reconnectInterval - Reconnect interval in ms (default: 5000)
 * @param {number} options.maxReconnectAttempts - Max reconnect attempts (default: 10)
 * @param {Function} options.onMessage - Message handler
 * @param {Function} options.onOpen - Connection open handler
 * @param {Function} options.onClose - Connection close handler
 * @param {Function} options.onError - Error handler
 * 
 * @returns {Object} WebSocket state and methods
 */
const useWebSocket = (url, options = {}) => {
  const {
    enabled = true,
    reconnectInterval = 5000,
    maxReconnectAttempts = 10,
    onMessage,
    onOpen,
    onClose,
    onError
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const [error, setError] = useState(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const shouldReconnectRef = useRef(true);

  // Send message through WebSocket
  const sendMessage = useCallback((data) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      try {
        const message = typeof data === 'string' ? data : JSON.stringify(data);
        wsRef.current.send(message);
        return true;
      } catch (err) {
        console.error('Failed to send WebSocket message:', err);
        setError(err);
        return false;
      }
    }
    return false;
  }, []);

  // Close WebSocket connection
  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (!url || !enabled) return;

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
    }

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = (event) => {
        console.log('WebSocket connected:', url);
        setIsConnected(true);
        setError(null);
        setReconnectAttempts(0);
        shouldReconnectRef.current = true;
        
        if (onOpen) {
          onOpen(event);
        }
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          setLastMessage(data);
          
          if (onMessage) {
            onMessage(data);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
          setLastMessage(event.data);
          
          if (onMessage) {
            onMessage(event.data);
          }
        }
      };

      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        const errorObj = new Error('WebSocket connection error');
        setError(errorObj);
        
        if (onError) {
          onError(errorObj);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
        setIsConnected(false);
        wsRef.current = null;
        
        if (onClose) {
          onClose(event);
        }

        // Auto-reconnect logic
        if (shouldReconnectRef.current && reconnectAttempts < maxReconnectAttempts) {
          console.log(`Reconnecting in ${reconnectInterval}ms... (attempt ${reconnectAttempts + 1}/${maxReconnectAttempts})`);
          
          reconnectTimeoutRef.current = setTimeout(() => {
            setReconnectAttempts(prev => prev + 1);
            connect();
          }, reconnectInterval);
        } else if (reconnectAttempts >= maxReconnectAttempts) {
          console.error('Max reconnect attempts reached');
          setError(new Error('Max reconnect attempts reached'));
        }
      };
    } catch (err) {
      console.error('Failed to create WebSocket connection:', err);
      setError(err);
    }
  }, [url, enabled, reconnectInterval, maxReconnectAttempts, reconnectAttempts, onMessage, onOpen, onClose, onError]);

  // Initialize connection
  useEffect(() => {
    if (enabled && url) {
      connect();
    }

    // Cleanup on unmount
    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [url, enabled, connect]);

  return {
    isConnected,
    lastMessage,
    error,
    reconnectAttempts,
    sendMessage,
    disconnect,
    reconnect: connect
  };
};

export default useWebSocket;
