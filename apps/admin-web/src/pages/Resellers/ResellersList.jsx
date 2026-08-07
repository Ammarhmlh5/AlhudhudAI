import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Handshake, Search, Plus, X, Globe, ExternalLink, Trash2, Activity, ShieldCheck, Wallet
} from 'lucide-react';
import apiClient from '../../api/axios';
import { RESELLER_PORTAL_URL } from '../../config';

export default function ResellersList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [resellers, setResellers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newReseller, setNewReseller] = useState({
    business_name: '',
    contact_name: '',
    email: '',
    phone: '',
    country_code: 'YE',
    password: ''
  });

  useEffect(() => {
    fetchResellers();
  }, []);

  const fetchResellers = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/api/v1/admin/resellers');
      setResellers(response.data || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || t('resellers.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  const handleCreateReseller = async (e) => {
    e.preventDefault();
    try {
      setCreating(true);
      await apiClient.post('/api/v1/admin/tenants', { ...newReseller, is_reseller: true });
      setShowCreateModal(false);
      setNewReseller({ business_name: '', contact_name: '', email: '', phone: '', country_code: 'YE', password: '' });
      fetchResellers();
    } catch (err) {
      alert(err.response?.data?.error || t('resellers.create_failed'));
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteReseller = async (id) => {
    if (!window.confirm(t('resellers.delete_confirm'))) return;
    try {
      await apiClient.post('/api/v1/admin/tenants/bulk-delete', { ids: [id] });
      fetchResellers();
    } catch (err) {
      alert(err.response?.data?.error || t('resellers.delete_failed'));
    }
  };

  const filtered = resellers.filter(r =>
    !searchTerm ||
    (r.business_name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (r.contact_name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (r.email || '').toLowerCase().includes(searchTerm.toLowerCase())
  );

  const formatDate = (iso) => {
    if (!iso) return '—';
    try {
      return new Date(iso).toLocaleDateString();
    } catch {
      return '—';
    }
  };

  const formatBalance = (value) =>
    Number(value || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-indigo-600 rounded-2xl shadow-lg shadow-indigo-900/20">
              <Handshake className="w-8 h-8 text-white" />
            </div>
            {t('resellers.title')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">
            {t('resellers.description')}
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-indigo-900/20"
          >
            <Plus className="w-4 h-4" />
            {t('resellers.add')}
          </button>
          <div className="bg-slate-900 border border-slate-800 p-1.5 rounded-2xl flex items-center gap-2">
            <div className="px-4 py-2 bg-slate-950 rounded-xl border border-slate-800 text-sm font-bold text-white">
              {resellers.length} {t('resellers.registered')}
            </div>
          </div>
        </div>
      </div>

      {/* Create Reseller Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-2xl w-full shadow-2xl relative overflow-hidden">
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-indigo-600 via-rose-600 to-indigo-600"></div>

            <div className="flex justify-between items-center mb-8">
              <div>
                <h2 className="text-3xl font-black text-white tracking-tight">{t('resellers.add')}</h2>
                <p className="text-slate-500 text-sm font-bold uppercase tracking-widest mt-1">{t('resellers.onboard')}</p>
              </div>
              <button onClick={() => setShowCreateModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                <X className="w-6 h-6" />
              </button>
            </div>

            <form onSubmit={handleCreateReseller} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.company_name')}</label>
                  <input
                    type="text"
                    required
                    value={newReseller.business_name}
                    onChange={e => setNewReseller({ ...newReseller, business_name: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold"
                    placeholder={t('resellers.name_placeholder')}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.contact_name')}</label>
                  <input
                    type="text"
                    required
                    value={newReseller.contact_name}
                    onChange={e => setNewReseller({ ...newReseller, contact_name: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold"
                    placeholder={t('resellers.contact_placeholder')}
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.email_address')}</label>
                  <input
                    type="email"
                    required
                    value={newReseller.email}
                    onChange={e => setNewReseller({ ...newReseller, email: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold"
                    placeholder={t('resellers.email_placeholder')}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.phone_number')}</label>
                  <input
                    type="text"
                    required
                    value={newReseller.phone}
                    onChange={e => setNewReseller({ ...newReseller, phone: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold"
                    placeholder="77XXXXXXX"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.country_code')}</label>
                  <select
                    value={newReseller.country_code}
                    onChange={e => setNewReseller({ ...newReseller, country_code: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold appearance-none cursor-pointer"
                  >
                    <option value="YE">Yemen (+967)</option>
                    <option value="SA">Saudi Arabia (+966)</option>
                    <option value="AE">UAE (+971)</option>
                    <option value="EG">Egypt (+20)</option>
                    <option value="JO">Jordan (+962)</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('resellers.initial_password')}</label>
                  <input
                    type="password"
                    required
                    value={newReseller.password}
                    onChange={e => setNewReseller({ ...newReseller, password: e.target.value })}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-indigo-500 transition-all font-bold"
                    placeholder="Min 8 characters"
                  />
                </div>
              </div>

              <div className="flex gap-4 pt-6">
                <button
                  type="submit"
                  disabled={creating}
                  className="flex-1 bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-indigo-900/20"
                >
                  {creating ? t('resellers.initializing') : t('resellers.confirm_onboarding')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Search Bar */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        <div className="flex-1 relative group">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-indigo-500 transition-colors" />
          <input
            type="text"
            placeholder={t('resellers.search_placeholder')}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-12 pr-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white placeholder:text-slate-600 focus:outline-none focus:border-indigo-500 transition-all shadow-inner"
          />
        </div>
      </div>

      {/* Resellers Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-32 space-y-4">
            <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin"></div>
            <p className="text-slate-400 font-bold animate-pulse">{t('resellers.syncing')}</p>
          </div>
        ) : error ? (
          <div className="p-12 text-center">
            <div className="bg-rose-500/10 text-rose-400 border border-rose-500/20 rounded-2xl p-6 inline-block">
              <p className="font-bold">{error}</p>
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-right border-collapse">
              <thead>
                <tr className="bg-slate-950/50 border-b border-slate-800">
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.software_company')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.contact')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.connected_companies')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.wallet')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.status')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('resellers.joined')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {filtered.map((r) => (
                  <tr
                    key={r.id}
                    onClick={() => navigate(`/resellers/${r.id}`)}
                    className="hover:bg-slate-800/30 transition-all group border-slate-800/50 cursor-pointer"
                  >
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 bg-slate-950 rounded-2xl flex items-center justify-center border border-slate-800 group-hover:border-indigo-500/30 transition-colors">
                          <span className="text-lg font-black text-slate-500 group-hover:text-indigo-500 transition-colors">
                            {(r.business_name || 'R').charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div>
                          <div className="text-base font-bold text-white group-hover:text-indigo-500 transition-colors">
                            {r.business_name}
                          </div>
                          <div className="text-xs text-slate-500 font-mono mt-0.5">
                            {r.company_serial || '—'}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="text-sm font-bold text-slate-300">{r.contact_name || '—'}</div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5">{r.email || '—'}</div>
                      <div className="text-xs text-slate-600 font-mono mt-0.5">{r.phone || '—'}</div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-3">
                        <Activity className="w-4 h-4 text-slate-600" />
                        <span className="text-sm font-bold text-slate-300 font-mono">
                          {(r.connected_companies || 0).toLocaleString()}
                        </span>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-3">
                        <Wallet className="w-4 h-4 text-emerald-500" />
                        <span className="text-sm font-bold text-slate-300 font-mono">
                          {formatBalance(r.balance)}
                        </span>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <span className={`px-3 py-1.5 text-[10px] font-black uppercase rounded-xl border flex items-center gap-1.5 w-fit ${
                        r.is_active
                          ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                          : 'bg-slate-500/10 text-slate-400 border-slate-500/20'
                      }`}>
                        <ShieldCheck className="w-3 h-3" />
                        {r.is_active ? t('resellers.active') : t('resellers.inactive')}
                      </span>
                    </td>
                    <td className="px-8 py-6 text-sm font-bold text-slate-400">
                      {formatDate(r.created_at)}
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex gap-2 justify-end">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            window.open(RESELLER_PORTAL_URL, '_blank', 'noopener,noreferrer');
                          }}
                          className="p-2 bg-slate-950 text-indigo-500 hover:text-white hover:bg-indigo-600 rounded-xl transition-all border border-slate-800"
                          title={t('resellers.open_portal')}
                        >
                          <ExternalLink className="w-5 h-5" />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteReseller(r.id);
                          }}
                          className="p-2 bg-slate-950 text-rose-500 hover:text-white hover:bg-rose-600 rounded-xl transition-all border border-slate-800"
                          title={t('resellers.delete')}
                        >
                          <Trash2 className="w-5 h-5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {filtered.length === 0 && !loading && (
        <div className="flex flex-col items-center justify-center py-24 bg-slate-900/50 border border-slate-800 border-dashed rounded-3xl">
          <Handshake className="w-16 h-16 text-slate-800 mb-4" />
          <p className="text-slate-500 font-bold text-lg">{t('resellers.no_companies')}</p>
          <button onClick={() => setSearchTerm('')} className="mt-4 text-indigo-500 font-bold hover:underline">
            {t('resellers.reset_search')}
          </button>
        </div>
      )}

      {/* Reseller Portal Note */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 flex items-center gap-4">
        <Globe className="w-6 h-6 text-indigo-500" />
        <div>
          <div className="text-sm font-black text-white">{t('resellers.portal')}</div>
          <div className="text-xs text-slate-500 font-mono mt-0.5">
            {t('resellers.portal_note', { url: RESELLER_PORTAL_URL })}
          </div>
        </div>
      </div>
    </div>
  );
}
