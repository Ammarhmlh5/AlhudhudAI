import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  ArrowLeft, Handshake, Wallet, Building2, X, ShieldCheck,
  User, Mail, Phone, Globe, Activity, KeyRound, Calendar
} from 'lucide-react';
import apiClient from '../../api/axios';

export default function ResellerDetails() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const [reseller, setReseller] = useState(null);
  const [companies, setCompanies] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [showFundModal, setShowFundModal] = useState(false);
  const [funding, setFunding] = useState(false);
  const [fundAmount, setFundAmount] = useState('');
  const [fundReason, setFundReason] = useState('');

  const fetchDetail = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get(`/api/v1/admin/resellers/${id}`);
      setReseller(response.data?.reseller || null);
      setCompanies(response.data?.companies || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || t('resellers.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const handleFundWallet = async (e) => {
    e.preventDefault();
    try {
      setFunding(true);
      const amount = Number(fundAmount);
      if (!amount || amount <= 0) {
        alert(t('resellers.detail.invalid_amount'));
        return;
      }
      await apiClient.post(`/api/v1/admin/resellers/${id}/credit`, {
        amount,
        reason: fundReason || undefined
      });
      setShowFundModal(false);
      setFundAmount('');
      setFundReason('');
      await fetchDetail();
    } catch (err) {
      alert(err.response?.data?.error || t('resellers.detail.fund_failed'));
    } finally {
      setFunding(false);
    }
  };

  const formatDate = (iso) => {
    if (!iso) return '—';
    try {
      return new Date(iso).toLocaleDateString();
    } catch {
      return '—';
    }
  };

  const subscriptionBadge = (sub) => {
    const map = {
      active: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
      grace_period: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
      none: 'bg-slate-500/10 text-slate-400 border-slate-500/20'
    };
    const label = {
      active: t('resellers.detail.sub_active'),
      grace_period: t('resellers.detail.sub_grace'),
      none: t('resellers.detail.sub_none')
    };
    return (
      <span className={`px-3 py-1.5 text-[10px] font-black uppercase rounded-xl border w-fit ${map[sub] || map.none}`}>
        {label[sub] || label.none}
      </span>
    );
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center py-32 space-y-4">
        <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin"></div>
        <p className="text-slate-400 font-bold animate-pulse">{t('resellers.syncing')}</p>
      </div>
    );
  }

  if (error && !reseller) {
    return (
      <div className="p-8">
        <div className="p-12 text-center">
          <div className="bg-rose-500/10 text-rose-400 border border-rose-500/20 rounded-2xl p-6 inline-block">
            <p className="font-bold">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  if (!reseller) {
    return (
      <div className="p-8">
        <div className="text-center py-24">
          <Handshake className="w-16 h-16 text-slate-800 mx-auto mb-4" />
          <p className="text-slate-500 font-bold">{t('resellers.detail.not_found')}</p>
          <Link to="/resellers" className="inline-block mt-4 text-indigo-500 font-bold hover:underline">
            {t('resellers.detail.back')}
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <button
            onClick={() => navigate('/resellers')}
            className="flex items-center gap-2 text-slate-400 hover:text-white font-bold text-sm mb-4 transition-all group"
          >
            <ArrowLeft className="w-4 h-4 group-rtl:rotate-180 rtl:rotate-180" />
            {t('resellers.detail.back')}
          </button>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-indigo-600 rounded-2xl shadow-lg shadow-indigo-900/20">
              <Handshake className="w-8 h-8 text-white" />
            </div>
            {reseller.business_name}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">
            {reseller.company_serial || '—'}
          </p>
        </div>

        <button
          onClick={() => setShowFundModal(true)}
          className="px-6 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-emerald-900/20"
        >
          <Wallet className="w-4 h-4" />
          {t('resellers.detail.fund_wallet')}
        </button>
      </div>

      {/* Fund Wallet Modal */}
      {showFundModal && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-xl w-full shadow-2xl relative overflow-hidden">
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-emerald-600 via-emerald-500 to-emerald-600"></div>

            <div className="flex justify-between items-center mb-8">
              <div>
                <h2 className="text-3xl font-black text-white tracking-tight">{t('resellers.detail.fund_title')}</h2>
                <p className="text-slate-500 text-sm font-bold uppercase tracking-widest mt-1">
                  {reseller.business_name}
                </p>
              </div>
              <button onClick={() => setShowFundModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                <X className="w-6 h-6" />
              </button>
            </div>

            <form onSubmit={handleFundWallet} className="space-y-6">
              <div className="space-y-2">
                <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.detail.amount')}</label>
                <input
                  type="number"
                  step="0.01"
                  min="0.01"
                  required
                  value={fundAmount}
                  onChange={e => setFundAmount(e.target.value)}
                  className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-emerald-500 transition-all font-bold"
                  placeholder="0.00"
                />
              </div>
              <div className="space-y-2">
                <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.detail.reason')}</label>
                <input
                  type="text"
                  value={fundReason}
                  onChange={e => setFundReason(e.target.value)}
                  className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-emerald-500 transition-all font-bold"
                  placeholder={t('resellers.detail.reason_placeholder')}
                />
              </div>

              <div className="flex gap-4 pt-6">
                <button
                  type="button"
                  onClick={() => setShowFundModal(false)}
                  className="flex-1 bg-slate-800 hover:bg-slate-700 text-slate-300 py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all"
                >
                  {t('resellers.detail.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={funding}
                  className="flex-1 bg-emerald-600 hover:bg-emerald-500 disabled:bg-emerald-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-emerald-900/20"
                >
                  {funding ? t('resellers.detail.funding') : t('resellers.detail.confirm_funding')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="p-8 bg-slate-900 border border-slate-800 rounded-[2rem] shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-2xl text-emerald-400">
              <Wallet className="w-6 h-6" />
            </div>
          </div>
          <div className="text-4xl font-black text-white tracking-tighter mb-2">
            {Number(reseller.balance || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <p className="text-[10px] font-black uppercase tracking-[0.3em] text-slate-500">{t('resellers.detail.wallet_balance')}</p>
        </div>

        <div className="p-8 bg-slate-900 border border-slate-800 rounded-[2rem] shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 bg-indigo-500/10 border border-indigo-500/20 rounded-2xl text-indigo-400">
              <Building2 className="w-6 h-6" />
            </div>
          </div>
          <div className="text-4xl font-black text-white tracking-tighter mb-2">
            {companies.length.toLocaleString()}
          </div>
          <p className="text-[10px] font-black uppercase tracking-[0.3em] text-slate-500">{t('resellers.detail.companies_count')}</p>
        </div>

        <div className="p-8 bg-slate-900 border border-slate-800 rounded-[2rem] shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-2xl text-rose-400">
              <ShieldCheck className="w-6 h-6" />
            </div>
          </div>
          <div className="flex items-center gap-3 mt-3 mb-2">
            <span className={`px-3 py-1.5 text-[10px] font-black uppercase rounded-xl border flex items-center gap-1.5 w-fit ${
              reseller.is_active
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                : 'bg-slate-500/10 text-slate-400 border-slate-500/20'
            }`}>
              {reseller.is_active ? t('resellers.active') : t('resellers.inactive')}
            </span>
          </div>
          <p className="text-[10px] font-black uppercase tracking-[0.3em] text-slate-500">{t('resellers.status')}</p>
        </div>
      </div>

      {/* Reseller Info */}
      <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 shadow-2xl">
        <h3 className="text-xl font-black text-white mb-6 flex items-center gap-4">
          <div className="p-2 bg-indigo-500/20 rounded-lg text-indigo-500">
            <User className="w-5 h-5" />
          </div>
          {t('resellers.detail.contact_info')}
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <User className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.detail.contact_name')}</div>
              <div className="text-sm font-bold text-white mt-1">{reseller.contact_name || '—'}</div>
            </div>
          </div>
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <Mail className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.email_address')}</div>
              <div className="text-sm font-bold text-white mt-1 font-mono">{reseller.email || '—'}</div>
            </div>
          </div>
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <Phone className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.phone_number')}</div>
              <div className="text-sm font-bold text-white mt-1 font-mono">{reseller.phone || '—'}</div>
            </div>
          </div>
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <Globe className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.country_code')}</div>
              <div className="text-sm font-bold text-white mt-1">{reseller.country_code || '—'}</div>
            </div>
          </div>
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <KeyRound className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.company_serial')}</div>
              <div className="text-sm font-bold text-white mt-1 font-mono">{reseller.company_serial || '—'}</div>
            </div>
          </div>
          <div className="p-5 bg-slate-950 border border-slate-800 rounded-2xl flex items-center gap-4">
            <Calendar className="w-5 h-5 text-slate-600 shrink-0" />
            <div>
              <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('resellers.joined')}</div>
              <div className="text-sm font-bold text-white mt-1">{formatDate(reseller.created_at)}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Customer Companies Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
        <div className="px-8 py-6 border-b border-slate-800 flex items-center justify-between">
          <h3 className="text-xl font-black text-white flex items-center gap-4">
            <div className="p-2 bg-indigo-500/20 rounded-lg text-indigo-500">
              <Activity className="w-5 h-5" />
            </div>
            {t('resellers.detail.customer_companies')}
          </h3>
          <div className="px-4 py-2 bg-slate-950 rounded-xl border border-slate-800 text-sm font-bold text-white">
            {companies.length} {t('resellers.registered')}
          </div>
        </div>

        {companies.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20">
            <Handshake className="w-16 h-16 text-slate-800 mb-4" />
            <p className="text-slate-500 font-bold text-lg">{t('resellers.detail.no_customers')}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-right border-collapse">
              <thead>
                <tr className="bg-slate-950/50 border-b border-slate-800">
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.software_company')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.contact')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.detail.wallet')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.detail.subscription')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.detail.api_key')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {companies.map((c) => (
                  <tr key={c.id} className="hover:bg-slate-800/30 transition-all group border-slate-800/50">
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 bg-slate-950 rounded-2xl flex items-center justify-center border border-slate-800 group-hover:border-indigo-500/30 transition-colors">
                          <span className="text-lg font-black text-slate-500 group-hover:text-indigo-500 transition-colors">
                            {(c.business_name || 'C').charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div>
                          <div className="text-base font-bold text-white group-hover:text-indigo-500 transition-colors">
                            {c.business_name}
                          </div>
                          <div className="text-xs text-slate-500 font-mono mt-0.5">
                            {c.company_serial || '—'}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="text-sm font-bold text-slate-300">{c.contact_name || '—'}</div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5">{c.email || '—'}</div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-3">
                        <Wallet className="w-4 h-4 text-emerald-500" />
                        <span className="text-sm font-bold text-slate-300 font-mono">
                          {Number(c.balance || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                        </span>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      {subscriptionBadge(c.subscription)}
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-2">
                        <KeyRound className="w-4 h-4 text-slate-600 shrink-0" />
                        <span className="text-xs font-bold text-slate-400 font-mono">
                          {c.api_key_accounting
                            ? `${c.api_key_accounting.slice(0, 8)}••••••••`
                            : '—'}
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
