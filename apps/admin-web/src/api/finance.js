import apiClient from './axios';

/**
 * Finance Operations API Client
 * Connects to the Finance Microservice
 */

// ==================== Transactions ====================

/**
 * Get transaction history
 * GET /api/finance/transactions
 */
export const getTransactions = (params = {}) => {
  return apiClient.get('/api/finance/transactions', { params });
};

/**
 * Create receipt voucher (شحن رصيد)
 * POST /api/finance/charge
 */
export const createReceipt = (data) => {
  return apiClient.post('/api/finance/charge', data);
};

/**
 * Create payment voucher (سند صرف)
 * POST /api/finance/debit-cashbox
 */
export const createPayment = (data) => {
  return apiClient.post('/api/finance/debit-cashbox', data);
};

/**
 * Reverse a transaction
 * POST /api/finance/reverse
 */
export const reverseTransaction = (data) => {
  return apiClient.post('/api/finance/reverse', data);
};

// ==================== Cashboxes ====================

/**
 * Get all cashboxes
 * GET /api/finance/cashboxes
 */
export const getCashboxes = () => {
  return apiClient.get('/api/finance/cashboxes');
};

/**
 * Create new cashbox
 * POST /api/finance/cashboxes
 */
export const createCashbox = (data) => {
  return apiClient.post('/api/finance/cashboxes', data);
};

/**
 * Get cashbox statement/statement
 * GET /api/finance/statement
 */
export const getCashboxStatement = (params) => {
  return apiClient.get('/api/finance/statement', { params });
};

// ==================== Wallets & Balance ====================

/**
 * Get wallet balance for a company/tenant
 * GET /api/finance/balance
 */
export const getBalance = (params) => {
  return apiClient.get('/api/finance/balance', { params });
};

/**
 * Charge wallet (شحن محفظة)
 * POST /api/finance/charge
 */
export const chargeWallet = (data) => {
  return apiClient.post('/api/finance/charge', data);
};

// ==================== Invoices ====================

/**
 * Get invoices list
 * GET /api/finance/invoices
 */
export const getInvoices = (params = {}) => {
  return apiClient.get('/api/finance/invoices', { params });
};

/**
 * Get invoice HTML for printing
 * GET /api/finance/invoices/print
 */
export const getInvoicePrint = (params) => {
  return apiClient.get('/api/finance/invoices/print', { params });
};

// ==================== Packages ====================

/**
 * Purchase a package
 * POST /api/finance/purchase
 */
export const purchasePackage = (data) => {
  return apiClient.post('/api/finance/purchase', data);
};

// ==================== Wallets ====================

/**
 * Create wallet for new tenant/reseller
 * POST /api/finance/wallets
 */
export const createWallet = (data) => {
  return apiClient.post('/api/finance/wallets', data);
};

// ==================== Reports ====================

/**
 * Get revenue report
 * GET /api/v1/admin/reports
 */
export const getRevenueReport = (params = {}) => {
  return apiClient.get('/api/v1/admin/reports', { params });
};

/**
 * Export report
 * GET /api/v1/admin/reports/export
 */
export const exportReport = (params = {}) => {
  return apiClient.get('/api/v1/admin/reports/export', { params, responseType: 'blob' });
};

// Export all functions as a single object
const financeAPI = {
  // Transactions
  getTransactions,
  createReceipt,
  createPayment,
  reverseTransaction,

  // Cashboxes
  getCashboxes,
  createCashbox,
  getCashboxStatement,

  // Balance & Wallet
  getBalance,
  chargeWallet,

  // Invoices
  getInvoices,
  getInvoicePrint,

  // Packages
  purchasePackage,

  // Wallets
  createWallet,

  // Reports
  getRevenueReport,
  exportReport,
};

export default financeAPI;