import apiClient from './axios';

export interface SendMessageRequest {
  to: string;
  content: string;
  channel: string;
}

export interface SendMessageResponse {
  message_id: string;
  status: string;
}

export interface TenantStats {
  totalTenants: number;
  activeTenants: number;
  messagesToday: number;
  revenue: number;
}

export async function getTenantStats(): Promise<TenantStats> {
  const response = await apiClient.get('/api/v1/admin/stats/tenants');
  return response.data;
}

export async function sendMessage(request: SendMessageRequest): Promise<SendMessageResponse> {
  const response = await apiClient.post('/api/v1/messages/send', request);
  return response.data;
}

export { default as apiClient } from './axios';
export { initializeCSRF, getCsrfToken, clearCsrfToken } from './axios';
