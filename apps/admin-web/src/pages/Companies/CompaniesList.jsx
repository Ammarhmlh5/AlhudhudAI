import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { 
  Building2, Search, Filter, ArrowUpDown, ChevronRight, 
  TrendingUp, Activity, Globe, Calendar, X,
  ExternalLink, MoreHorizontal, Plus, Trash2
} from 'lucide-react';
import apiClient from '../../api/axios';

export default function CompaniesList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [companies, setCompanies] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filters, setFilters] = useState({
    rating_category: '',
    date_from: '',
    date_to: '',
  });
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState('desc');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const itemsPerPage = 20;
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newCompany, setNewCompany] = useState({
    business_name: '',
    contact_name: '',
    email: '',
    phone: '',
    country_code: 'YE',
    password: ''
  });

  useEffect(() => {
    fetchCompanies();
  }, [searchTerm, filters, sortBy, sortOrder, currentPage]);

  const fetchCompanies = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams({
        page: currentPage,
        page_size: itemsPerPage,
        search: searchTerm,
      });

      if (filters.rating_category) params.append('rating_category', filters.rating_category);
      if (filters.date_from) params.append('date_from', filters.date_from);
      if (filters.date_to) params.append('date_to', filters.date_to);

      const response = await apiClient.get(`/api/v1/admin/tenants?${params}`);

      const mappedCompanies = (response.data.tenants || []).map(t => ({
        tenant_id: t.id,
        tenant_name: t.business_name || t.contact_name,
        overall_rating: t.rating || 0,
        rating_category: t.rating_category || 'average',
        sending_volume: t.sending_volume || 0,
        success_rate: t.success_rate || 0,
        created_at: t.created_at,
      }));

      setCompanies(mappedCompanies);
      setTotalPages(Math.ceil((response.data.total_count || 0) / itemsPerPage));
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to load companies');
    } finally {
      setLoading(false);
    }
  };

  const getRatingBadgeStyles = (category) => {
    switch (category) {
      case 'excellent': return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
      case 'good':      return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
      case 'average':   return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
      case 'poor':      return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
      default:          return 'bg-slate-500/10 text-slate-400 border-slate-500/20';
    }
  };

  const handleSort = (field) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('desc');
    }
  };

  const handleCreateCompany = async (e) => {
    e.preventDefault();
    try {
        setCreating(true);
        await apiClient.post('/api/v1/admin/tenants', newCompany);
        setShowCreateModal(false);
        setNewCompany({ business_name: '', contact_name: '', email: '', phone: '', country_code: 'YE', password: '' });
        fetchCompanies();
    } catch (err) {
        alert(err.response?.data?.error || 'Failed to create company');
    } finally {
        setCreating(false);
    }
  };

  const handleDeleteCompanyRow = async (id) => {
    if (!window.confirm('Are you sure you want to delete this company?')) return;
    try {
        await apiClient.post('/api/v1/admin/tenants/bulk-delete', { ids: [id] });
        fetchCompanies();
    } catch (err) {
        alert(err.response?.data?.error || 'Failed to delete company');
    }
  };

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-rose-600 rounded-2xl shadow-lg shadow-rose-900/20">
              <Building2 className="w-8 h-8 text-white" />
            </div>
            {t('companies.title')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">
            {t('companies.description')}
          </p>
        </div>
        
        <div className="flex items-center gap-3">
            <button
                onClick={() => setShowCreateModal(true)}
                className="px-6 py-3 bg-rose-600 hover:bg-rose-700 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-rose-900/20"
            >
                <Plus className="w-4 h-4" />
                Add Company
            </button>
            <div className="bg-slate-900 border border-slate-800 p-1.5 rounded-2xl flex items-center gap-2">
                <div className="px-4 py-2 bg-slate-950 rounded-xl border border-slate-800 text-sm font-bold text-white">
                    {totalPages * itemsPerPage}+ Registered
                </div>
            </div>
        </div>
      </div>

      {/* Create Company Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
            <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-2xl w-full shadow-2xl relative overflow-hidden">
                <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-rose-600 via-indigo-600 to-rose-600"></div>
                
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <h2 className="text-3xl font-black text-white tracking-tight">Create Company</h2>
                        <p className="text-slate-500 text-sm font-bold uppercase tracking-widest mt-1">Onboard New Enterprise Tenant</p>
                    </div>
                    <button onClick={() => setShowCreateModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                        <X className="w-6 h-6" />
                    </button>
                </div>

                <form onSubmit={handleCreateCompany} className="space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Business Name</label>
                            <input
                                type="text"
                                required
                                value={newCompany.business_name}
                                onChange={e => setNewCompany({ ...newCompany, business_name: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold"
                                placeholder="e.g. Acme Corp"
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Contact Name</label>
                            <input
                                type="text"
                                required
                                value={newCompany.contact_name}
                                onChange={e => setNewCompany({ ...newCompany, contact_name: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold"
                                placeholder="John Doe"
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Email Address</label>
                            <input
                                type="email"
                                required
                                value={newCompany.email}
                                onChange={e => setNewCompany({ ...newCompany, email: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold"
                                placeholder="admin@company.com"
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Phone Number</label>
                            <input
                                type="text"
                                required
                                value={newCompany.phone}
                                onChange={e => setNewCompany({ ...newCompany, phone: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold"
                                placeholder="77XXXXXXX"
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Country Code</label>
                            <select
                                value={newCompany.country_code}
                                onChange={e => setNewCompany({ ...newCompany, country_code: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold appearance-none cursor-pointer"
                            >
                                <option value="YE">Yemen (+967)</option>
                                <option value="SA">Saudi Arabia (+966)</option>
                                <option value="AE">UAE (+971)</option>
                                <option value="EG">Egypt (+20)</option>
                                <option value="JO">Jordan (+962)</option>
                            </select>
                        </div>
                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">Initial Password</label>
                            <input
                                type="password"
                                required
                                value={newCompany.password}
                                onChange={e => setNewCompany({ ...newCompany, password: e.target.value })}
                                className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold"
                                placeholder="Min 8 characters"
                            />
                        </div>
                    </div>

                    <div className="flex gap-4 pt-6">
                        <button
                            type="submit"
                            disabled={creating}
                            className="flex-1 bg-rose-600 hover:bg-rose-500 disabled:bg-rose-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-rose-900/20"
                        >
                            {creating ? 'Initializing...' : 'Confirm Onboarding'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
      )}

      {/* Search and Filters Bar */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        <div className="flex flex-col lg:flex-row gap-6">
          {/* Search */}
          <div className="flex-1 relative group">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-rose-500 transition-colors" />
            <input
              type="text"
              placeholder={t('companies.search_placeholder')}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-12 pr-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white placeholder:text-slate-600 focus:outline-none focus:border-rose-500 transition-all shadow-inner"
            />
          </div>

          {/* Filters Group */}
          <div className="flex flex-wrap items-center gap-4">
            <div className="relative min-w-[160px]">
              <Filter className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
              <select
                value={filters.rating_category}
                onChange={(e) => setFilters({ ...filters, rating_category: e.target.value })}
                className="w-full pl-9 pr-8 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-sm text-slate-300 focus:outline-none focus:border-rose-500 appearance-none cursor-pointer"
              >
                <option value="">{t('companies.all_ratings')}</option>
                <option value="excellent">{t('companies.excellent')}</option>
                <option value="good">{t('companies.good')}</option>
                <option value="average">{t('companies.average')}</option>
                <option value="poor">{t('companies.poor')}</option>
              </select>
            </div>

            <div className="flex items-center gap-2 bg-slate-950 border border-slate-800 rounded-2xl px-3 py-1.5">
              <Calendar className="w-4 h-4 text-slate-500" />
              <input
                type="date"
                value={filters.date_from}
                onChange={(e) => setFilters({ ...filters, date_from: e.target.value })}
                className="bg-transparent text-xs text-slate-300 focus:outline-none"
              />
              <span className="text-slate-700">|</span>
              <input
                type="date"
                value={filters.date_to}
                onChange={(e) => setFilters({ ...filters, date_to: e.target.value })}
                className="bg-transparent text-xs text-slate-300 focus:outline-none"
              />
            </div>

            {(searchTerm || filters.rating_category || filters.date_from || filters.date_to) && (
                <button 
                    onClick={() => {
                        setSearchTerm('');
                        setFilters({ rating_category: '', date_from: '', date_to: '' });
                    }}
                    className="p-3 bg-rose-500/10 text-rose-500 rounded-2xl hover:bg-rose-500 hover:text-white transition-all"
                >
                    <X className="w-4 h-4" />
                </button>
            )}
          </div>
        </div>
      </div>

      {/* Companies Table Container */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
        {loading && companies.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-32 space-y-4">
            <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
            <p className="text-slate-400 font-bold animate-pulse">Syncing Company Data...</p>
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
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest cursor-pointer group" onClick={() => handleSort('name')}>
                    <div className="flex items-center gap-2">
                        {t('companies.company_name')}
                        <ArrowUpDown className={`w-3 h-3 transition-colors ${sortBy === 'name' ? 'text-rose-500' : 'text-slate-700 group-hover:text-slate-500'}`} />
                    </div>
                  </th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest cursor-pointer group" onClick={() => handleSort('rating')}>
                    <div className="flex items-center gap-2">
                        {t('companies.rating')}
                        <ArrowUpDown className={`w-3 h-3 transition-colors ${sortBy === 'rating' ? 'text-rose-500' : 'text-slate-700 group-hover:text-slate-500'}`} />
                    </div>
                  </th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('companies.category')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest cursor-pointer group" onClick={() => handleSort('messages')}>
                    <div className="flex items-center gap-2">
                        {t('companies.total_messages')}
                        <ArrowUpDown className={`w-3 h-3 transition-colors ${sortBy === 'messages' ? 'text-rose-500' : 'text-slate-700 group-hover:text-slate-500'}`} />
                    </div>
                  </th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('companies.success_rate')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {companies.map((company) => (
                  <tr
                    key={company.tenant_id}
                    className="hover:bg-slate-800/30 transition-all cursor-pointer group border-slate-800/50"
                    onClick={() => navigate(`/companies/${company.tenant_id}`)}
                  >
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 bg-slate-950 rounded-2xl flex items-center justify-center border border-slate-800 group-hover:border-rose-500/30 transition-colors">
                            <span className="text-lg font-black text-slate-500 group-hover:text-rose-500 transition-colors">
                                {company.tenant_name.charAt(0).toUpperCase()}
                            </span>
                        </div>
                        <div>
                            <div className="text-base font-bold text-white group-hover:text-rose-500 transition-colors">
                                {company.tenant_name}
                            </div>
                            <div className="text-xs text-slate-500 font-mono mt-0.5">
                                ID: {company.tenant_id.slice(0, 8)}...
                            </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-2">
                        <div className="text-lg font-black text-white">{company.overall_rating?.toFixed(1) || '—'}</div>
                        <div className="flex gap-0.5">
                            {[1, 2, 3, 4, 5].map(star => (
                                <div key={star} className={`w-1 h-3 rounded-full ${star <= company.overall_rating ? 'bg-amber-500' : 'bg-slate-800'}`} />
                            ))}
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <span className={`px-3 py-1.5 text-[10px] font-black uppercase rounded-xl border ${getRatingBadgeStyles(company.rating_category)}`}>
                        {t(`companies.${company.rating_category || 'unknown'}`)}
                      </span>
                    </td>
                    <td className="px-8 py-6">
                        <div className="flex items-center gap-3">
                            <Activity className="w-4 h-4 text-slate-600" />
                            <span className="text-sm font-bold text-slate-300 font-mono">
                                {company.sending_volume?.toLocaleString() || 0}
                            </span>
                        </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="w-32 space-y-1.5">
                        <div className="flex justify-between text-[10px] font-black uppercase">
                            <span className="text-slate-500">Success Rate</span>
                            <span className={company.success_rate > 90 ? 'text-emerald-400' : 'text-amber-400'}>{company.success_rate?.toFixed(1)}%</span>
                        </div>
                        <div className="h-1.5 bg-slate-950 rounded-full overflow-hidden border border-slate-800">
                            <div 
                                className={`h-full rounded-full transition-all duration-1000 ${company.success_rate > 90 ? 'bg-emerald-500' : 'bg-amber-500'}`}
                                style={{ width: `${company.success_rate}%` }}
                            />
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex gap-2 justify-end">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteCompanyRow(company.tenant_id);
                          }}
                          className="p-2 bg-slate-950 text-rose-500 hover:text-white hover:bg-rose-600 rounded-xl transition-all border border-slate-800"
                          title="Delete Company"
                        >
                          <Trash2 className="w-5 h-5" />
                        </button>
                        <button
                          className="p-2 bg-slate-950 text-slate-500 hover:text-white hover:bg-rose-600 rounded-xl transition-all border border-slate-800"
                          title="View Details"
                        >
                          <ChevronRight className="w-5 h-5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Premium Pagination */}
        {totalPages > 1 && (
          <div className="bg-slate-950/80 px-8 py-6 flex items-center justify-between border-t border-slate-800 backdrop-blur-md">
            <div className="text-xs font-bold text-slate-500 uppercase tracking-widest">
              Page <span className="text-white">{currentPage}</span> of <span className="text-white">{totalPages}</span>
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                disabled={currentPage === 1}
                className="px-6 py-2 bg-slate-900 border border-slate-800 rounded-xl text-xs font-black text-slate-400 hover:bg-slate-800 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-all uppercase"
              >
                {t('companies.previous')}
              </button>
              <button
                onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                disabled={currentPage === totalPages}
                className="px-6 py-2 bg-rose-600 rounded-xl text-xs font-black text-white hover:bg-rose-500 disabled:opacity-30 disabled:cursor-not-allowed transition-all uppercase shadow-lg shadow-rose-900/20"
              >
                {t('companies.next')}
              </button>
            </div>
          </div>
        )}
      </div>

      {companies.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center py-24 bg-slate-900/50 border border-slate-800 border-dashed rounded-3xl">
              <Globe className="w-16 h-16 text-slate-800 mb-4" />
              <p className="text-slate-500 font-bold text-lg">No companies found in this galaxy.</p>
              <button onClick={() => { setSearchTerm(''); setFilters({ rating_category: '', date_from: '', date_to: '' }); }} className="mt-4 text-rose-500 font-bold hover:underline">
                  Reset all filters
              </button>
          </div>
      )}
    </div>
  );
}

