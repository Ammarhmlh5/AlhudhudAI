import apiClient from './axios';

/**
 * Finance Service API Client
 * Connects to the Finance Microservice
 */

const FINANCE_API_URL = import.meta.env.VITE_FINANCE_API_URL || 'http://localhost:8081';

/**
 * Get invoices for current tenant
 * GET /api/finance/invoices
 */
export const getInvoices = (params = {}) => {
  return fetch(`${FINANCE_API_URL}/api/finance/invoices?${new URLSearchParams(params)}`, {
    headers: {
      'Content-Type': 'application/json',
    },
  }).then(res => res.json());
};

/**
 * Get invoice HTML for printing
 * GET /api/finance/invoices/print?id=...
 */
export const getInvoicePrint = (invoiceId) => {
  return fetch(`${FINANCE_API_URL}/api/finance/invoices/print?id=${invoiceId}`, {
    headers: {
      'Content-Type': 'text/html',
    },
  }).then(res => res.text());
};

/**
 * Get wallet balance for a tenant
 * GET /api/finance/balance?id=...
 */
export const getBalance = (walletId) => {
  return fetch(`${FINANCE_API_URL}/api/finance/balance?id=${walletId}`, {
    headers: {
      'Content-Type': 'application/json',
    },
  }).then(res => res.json());
};

// Export all functions as a single object
const financeAPI = {
  getInvoices,
  getInvoicePrint,
  getBalance,
};

export default financeAPI;