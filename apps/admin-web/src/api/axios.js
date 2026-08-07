// Import from shared library
import { createApiClient, initializeCSRF, getCsrfToken, clearCsrfToken } from '@alhudhud/shared/api';
import { API_BASE_URL } from '../config';

// Create API client using shared library
const apiClient = createApiClient({
  baseURL: API_BASE_URL,
  withCredentials: true,
});

// Re-export functions for backward compatibility
export { initializeCSRF, getCsrfToken, clearCsrfToken };

// Wrapper that passes the configured API_BASE_URL
export function initializeAppCSRF() {
    return initializeCSRF(API_BASE_URL);
}

export default apiClient;
