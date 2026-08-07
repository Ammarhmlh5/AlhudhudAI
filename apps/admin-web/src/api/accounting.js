import apiClient from './axios';

export const ACCOUNTING = {
  listCashboxes: () => apiClient.get('/api/v1/admin/accounting/cashboxes'),
  createCashbox: (data) => apiClient.post('/api/v1/admin/accounting/cashboxes', data),
  getCashbox: (id) => apiClient.get(`/api/v1/admin/accounting/cashboxes/${id}`),
  updateCashbox: (id, data) => apiClient.put(`/api/v1/admin/accounting/cashboxes/${id}`, data),
  setCashboxStatus: (id, status) => apiClient.put(`/api/v1/admin/accounting/cashboxes/${id}/status`, { status }),
  cashboxStatement: (id, data) => apiClient.post(`/api/v1/admin/accounting/cashboxes/${id}/statement`, data || {}),

  listReceipts: (params) => apiClient.get('/api/v1/admin/accounting/receipts', { params }),
  createReceipt: (data) => apiClient.post('/api/v1/admin/accounting/receipts', data),
  getReceipt: (id) => apiClient.get(`/api/v1/admin/accounting/receipts/${id}`),
  postReceipt: (id) => apiClient.post(`/api/v1/admin/accounting/receipts/${id}/post`),
  reverseReceipt: (id) => apiClient.post(`/api/v1/admin/accounting/receipts/${id}/reverse`),

  listPayments: (params) => apiClient.get('/api/v1/admin/accounting/payments', { params }),
  createPayment: (data) => apiClient.post('/api/v1/admin/accounting/payments', data),
  getPayment: (id) => apiClient.get(`/api/v1/admin/accounting/payments/${id}`),
  postPayment: (id) => apiClient.post(`/api/v1/admin/accounting/payments/${id}/post`),
  reversePayment: (id) => apiClient.post(`/api/v1/admin/accounting/payments/${id}/reverse`),

  previewNumber: (type) => apiClient.get(`/api/v1/admin/accounting/numbering/${type}`),
};
