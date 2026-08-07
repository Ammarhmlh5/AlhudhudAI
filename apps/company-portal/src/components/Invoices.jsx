import { useEffect, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { financeAPI } from '../api/finance';
import { useTranslation } from 'react-i18next';
import { Receipt, Download, Eye, FileText } from 'lucide-react';

function formatDate(value) {
  if (!value) return '—';
  return new Date(value).toLocaleDateString('ar-EG', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export default function Invoices() {
  const { user, tenant } = useAuth();
  const { t } = useTranslation();
  const [invoices, setInvoices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedInvoice, setSelectedInvoice] = useState(null);
  const [invoiceHtml, setInvoiceHtml] = useState('');

  useEffect(() => {
    loadInvoices();
  }, [tenant]);

  const loadInvoices = async () => {
    setLoading(true);
    try {
      const data = await financeAPI.getInvoices();
      setInvoices(data || []);
    } catch (err) {
      console.error('Failed to load invoices:', err);
      setInvoices([]);
    } finally {
      setLoading(false);
    }
  };

  const handleViewInvoice = async (invoiceId) => {
    try {
      const html = await financeAPI.getInvoicePrint(invoiceId);
      setInvoiceHtml(html);
      setSelectedInvoice(invoiceId);
    } catch (err) {
      console.error('Failed to load invoice:', err);
      alert(t('common.error'));
    }
  };

  const handlePrintInvoice = (invoiceId) => {
    window.open(
      `${import.meta.env.VITE_FINANCE_API_URL || 'http://localhost:8081'}/api/finance/invoices/print?id=${invoiceId}`,
      '_blank'
    );
  };

  return (
    <div className="min-h-screen bg-slate-950 pb-16">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 bg-rose-500/10 rounded-xl border border-rose-500/20">
              <FileText className="w-6 h-6 text-rose-500" />
            </div>
            <div>
              <h1 className="text-lg font-black text-white">{t('invoices.title')}</h1>
              <p className="text-xs text-slate-400">{t('invoices.description')}</p>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 mt-8">
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <div className="animate-spin rounded-full h-12 w-12 border-4 border-rose-500 border-t-transparent" />
          </div>
        ) : invoices.length === 0 ? (
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-12 text-center">
            <Receipt className="w-16 h-16 text-slate-600 mx-auto mb-4" />
            <h2 className="text-xl font-bold text-white mb-2">{t('invoices.no_invoices')}</h2>
            <p className="text-slate-400">{t('invoices.no_invoices_desc')}</p>
          </div>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-slate-800/50 text-slate-400">
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.invoice_number')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.customer')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.amount')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.vat')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.total')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.date')}</th>
                  <th className="text-right px-6 py-4 font-semibold">{t('invoices.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-slate-800/30 transition-colors">
                    <td className="px-6 py-4 font-mono text-rose-400 font-bold">
                      {inv.invoice_number}
                    </td>
                    <td className="px-6 py-4 text-white font-bold">
                      {inv.customer_name}
                    </td>
                    <td className="px-6 py-4 text-slate-300">
                      {inv.sub_total} {t('invoices.currency')}
                    </td>
                    <td className="px-6 py-4 text-slate-400">
                      {inv.vat_amount} {t('invoices.currency')}
                    </td>
                    <td className="px-6 py-4 text-emerald-400 font-bold">
                      {inv.total_amount} {t('invoices.currency')}
                    </td>
                    <td className="px-6 py-4 text-slate-400">
                      {formatDate(inv.issue_date)}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleViewInvoice(inv.id)}
                          className="p-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition-colors"
                          title={t('invoices.view')}
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handlePrintInvoice(inv.id)}
                          className="p-2 bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 rounded-lg transition-colors"
                          title={t('invoices.download')}
                        >
                          <Download className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Invoice Modal */}
        {selectedInvoice && invoiceHtml && (
          <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-3xl max-h-[90vh] overflow-hidden">
              <div className="flex items-center justify-between p-4 border-b border-slate-800">
                <h3 className="text-lg font-bold text-white">{t('invoices.title')}</h3>
                <button
                  onClick={() => {
                    setSelectedInvoice(null);
                    setInvoiceHtml('');
                  }}
                  className="p-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition-colors"
                >
                  {t('common.close')}
                </button>
              </div>
              <div className="overflow-auto max-h-[calc(90vh-80px)]">
                <div
                  className="p-8"
                  dangerouslySetInnerHTML={{ __html: invoiceHtml }}
                />
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}