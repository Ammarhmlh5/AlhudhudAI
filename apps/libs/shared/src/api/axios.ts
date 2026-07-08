import axios, { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse, AxiosError } from 'axios';

// Type definitions for API configuration
export interface ApiConfig {
  baseURL: string;
  withCredentials?: boolean;
  timeout?: number;
}

// Type definitions for CSRF token response
interface CsrfTokenResponse {
  csrf_token: string;
}

// Type definitions for rate limit error
interface RateLimitError extends Error {
  isRateLimitError: boolean;
  retryAfter: number;
  originalError: AxiosError;
}

// Type definitions for error response
interface ErrorResponse {
  message?: string;
  error?: {
    code?: string;
  };
  retry_after_seconds?: number;
}

// Default configuration
const defaultConfig: Partial<ApiConfig> = {
  withCredentials: true,
  timeout: 30000,
};

// CSRF token management state
let csrfToken: string | null = null;
let csrfFetchInProgress = false;
let lastRateLimitTime: number | null = null;

/**
 * Create and configure an Axios API client
 * @param config - API configuration options
 * @returns Configured Axios instance
 */
export function createApiClient(config: ApiConfig): AxiosInstance {
  const apiClient = axios.create({
    baseURL: config.baseURL,
    withCredentials: config.withCredentials ?? defaultConfig.withCredentials,
    timeout: config.timeout ?? defaultConfig.timeout,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Setup interceptors
  setupRequestInterceptor(apiClient);
  setupResponseInterceptor(apiClient);

  return apiClient;
}

/**
 * Fetch CSRF token from the server
 * @param baseURL - Base URL for API requests
 * @returns CSRF token string
 */
async function fetchCSRFToken(baseURL: string): Promise<string | null> {
  // Prevent multiple simultaneous CSRF token fetches
  if (csrfFetchInProgress) {
    console.log('CSRF token fetch already in progress, skipping...');
    return csrfToken;
  }

  // If we were rate limited recently (within last 60 seconds), don't retry
  if (lastRateLimitTime && Date.now() - lastRateLimitTime < 60000) {
    console.warn('Recently rate limited, skipping CSRF token fetch');
    return csrfToken;
  }

  csrfFetchInProgress = true;

  try {
    const response = await axios.get<CsrfTokenResponse>(`${baseURL}/api/v1/csrf-token`, {
      withCredentials: true,
    });

    if (response.data && response.data.csrf_token) {
      csrfToken = response.data.csrf_token;
      console.log('CSRF token fetched successfully');
      return csrfToken;
    }
    return null;
  } catch (error) {
    console.error('Failed to fetch CSRF token:', error);

    // Track rate limiting
    if ((error as AxiosError).response && (error as AxiosError).response?.status === 429) {
      lastRateLimitTime = Date.now();
    }

    throw error;
  } finally {
    csrfFetchInProgress = false;
  }
}

/**
 * Setup request interceptor to add CSRF token and auth headers
 */
function setupRequestInterceptor(apiClient: AxiosInstance): void {
  apiClient.interceptors.request.use(
    async (config: InternalAxiosRequestConfig) => {
      // Fallback to Bearer token from localStorage if available
      const token = localStorage.getItem('token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }

      // Add CSRF token for state-changing requests (POST, PUT, DELETE, PATCH)
      const method = config.method?.toUpperCase();
      if (method && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
        // If we don't have a CSRF token yet, fetch it
        if (!csrfToken) {
          try {
            const baseURL = config.baseURL || '';
            await fetchCSRFToken(baseURL);
          } catch (error) {
            console.error('Failed to fetch CSRF token before request:', error);
          }
        }

        // Add CSRF token to request header
        if (csrfToken) {
          config.headers['X-CSRF-Token'] = csrfToken;
        }
      }

      return config;
    },
    (error: AxiosError) => {
      return Promise.reject(error);
    }
  );
}

/**
 * Setup response interceptor to handle errors
 */
function setupResponseInterceptor(apiClient: AxiosInstance): void {
  apiClient.interceptors.response.use(
    (response: AxiosResponse) => response,
    async (error: AxiosError) => {
      const originalRequest = error.config as InternalAxiosRequestConfig & {
        _rateLimitRetry?: boolean;
        _csrfRetry?: boolean;
      };

      // Handle rate limiting errors (429 Too Many Requests)
      if (error.response && error.response.status === 429) {
        const retryAfter =
          error.response.headers['retry-after'] ||
          (error.response.data as ErrorResponse)?.retry_after_seconds ||
          60;

        console.warn(`Rate limit exceeded. Retry after ${retryAfter} seconds`);

        // Show user-friendly message
        const message =
          (error.response.data as ErrorResponse)?.message ||
          'Too many requests. Please slow down and try again in a moment.';

        console.error('Rate Limit Error:', message);

        // Reject with enhanced error information
        const enhancedError: RateLimitError = new Error(message) as RateLimitError;
        enhancedError.isRateLimitError = true;
        enhancedError.retryAfter = retryAfter;
        enhancedError.originalError = error;
        return Promise.reject(enhancedError);
      }

      // Handle CSRF token errors (403 with CSRF-related error codes)
      if (error.response && error.response.status === 403) {
        const errorData = error.response.data as ErrorResponse;
        const errorCode = errorData?.error?.code;

        // If CSRF token is invalid/expired/missing, refresh it and retry
        if (
          errorCode === 'CSRF_TOKEN_INVALID' ||
          errorCode === 'CSRF_TOKEN_MISSING' ||
          errorCode === 'CSRF_TOKEN_MISMATCH'
        ) {
          console.log('CSRF token error detected, refreshing token...');

          // Prevent infinite retry loop
          if (!originalRequest._csrfRetry) {
            originalRequest._csrfRetry = true;

            try {
              // Fetch new CSRF token
              const baseURL = originalRequest.baseURL || '';
              await fetchCSRFToken(baseURL);

              // Retry the original request with new token
              if (csrfToken) {
                originalRequest.headers['X-CSRF-Token'] = csrfToken;
                return apiClient(originalRequest);
              }
            } catch (refreshError) {
              console.error('Failed to refresh CSRF token:', refreshError);
              return Promise.reject(error);
            }
          }
        }
      }

      // Handle unauthorized errors (401)
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - potential session expiry');
        // Token is in httpOnly cookie (cleared by server on logout)
        // Redirect to login page if we aren't already there
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      }

      return Promise.reject(error);
    }
  );
}

/**
 * Initialize CSRF token (call this on app startup)
 * @param baseURL - Base URL for API requests
 */
export async function initializeCSRF(baseURL: string): Promise<void> {
  try {
    await fetchCSRFToken(baseURL);
  } catch (error) {
    console.error('Failed to initialize CSRF protection:', error);
    // Don't throw - allow app to continue even if CSRF init fails
    // CSRF token will be fetched on first state-changing request
  }
}

/**
 * Get current CSRF token
 * @returns Current CSRF token or null
 */
export function getCsrfToken(): string | null {
  return csrfToken;
}

/**
 * Clear CSRF token (useful on logout)
 */
export function clearCsrfToken(): void {
  csrfToken = null;
}

// Export default API client factory for convenience
export default createApiClient;