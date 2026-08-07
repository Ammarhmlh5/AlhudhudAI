import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Users, Search, Smartphone, Shield, Battery, Wifi, WifiOff, Wallet, Clock, CheckCircle, XCircle, ChevronRight, ArrowLeft, Key, Lock, Trash2, Plus, DollarSign, Download } from 'lucide-react';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

export default function TenantManager() {
    const { t } = useTranslation();
    const [view, setView] = useState('list'); // 'list' or 'detail'
    const [selectedTenant, setSelectedTenant] = useState(null);
    const [activeTab, setActiveTab] = useState('overview'); // for detail view: 'overview', 'devices', 'subscription', 'finance'
    const [showPaymentModal, setShowPaymentModal] = useState(false);
    const [tenants, setTenants] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [showAddModal, setShowAddModal] = useState(false);
    const [createLoading, setCreateLoading] = useState(false);
    const [newTenant, setNewTenant] = useState({
        business_name: '',
        contact_name: '',
        email: '',
        phone: '',
        country_code: 'SA',
        password: ''
    });

    const [page, setPage] = useState(1);
    const [pageSize] = useState(10);
    const [totalCount, setTotalCount] = useState(0);
    const [selectedIds, setSelectedIds] = useState([]);

    // Filtering state
    const [searchQuery, setSearchQuery] = useState('');
    const [debouncedSearch, setDebouncedSearch] = useState('');
    const [filters, setFilters] = useState({
        status: '',
        sector: '',
        minBalance: '',
        maxBalance: ''
    });

    // Detail data state
    const [subscriptionData, setSubscriptionData] = useState(null);
    const [transactions, setTransactions] = useState([]);
    const [detailLoading, setDetailLoading] = useState(false);

    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(searchQuery);
        }, 500);
        return () => clearTimeout(timer);
    }, [searchQuery]);

    useEffect(() => {
        fetchTenants(1); // Reset to page 1 on filter change
    }, [debouncedSearch, filters]);

    useEffect(() => {
        fetchTenants(page);
    }, [page]);

    const fetchTenants = async (currentPage) => {
        try {
            setLoading(true);
            let url = `${API_BASE_URL}/api/v1/admin/tenants?page=${currentPage}&page_size=${pageSize}`;
            if (debouncedSearch) url += `&search=${encodeURIComponent(debouncedSearch)}`;
            if (filters.status) url += `&status=${filters.status}`;
            if (filters.sector) url += `&sector=${filters.sector}`;
            if (filters.minBalance) url += `&min_balance=${filters.minBalance}`;
            if (filters.maxBalance) url += `&max_balance=${filters.maxBalance}`;

            const res = await apiClient.get(url);
            const data = res.data;

            setTotalCount(data.total_count);

            // Map backend data to frontend model requirements
            const mappedData = data.tenants.map(t => ({
                id: t.id,
                name: t.contact_name,
                business: t.business_name,
                email: t.email,
                phone: t.phone,
                plan: t.plan,
                status: t.status === 'ACTIVE' ? 'Active' : 'Suspended',
                balance: t.balance,
                devices: t.devices,
                joined: t.joined,
                last_login: 'N/A',
                has_dedicated_gateway: false,
                device_list: []
            }));

            setTenants(mappedData);
        } catch (err) {
            // Handled

            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const fetchSubscriptionData = async (tenantId) => {
        try {
            setDetailLoading(true);
            const res = await apiClient.get(`/api/v1/admin/tenants/${tenantId}/subscription`);
            setSubscriptionData(res.data);
        } catch (err) {
            console.error('Error fetching subscription:', err);
        } finally {
            setDetailLoading(false);
        }
    };

    const fetchTransactions = async (tenantId) => {
        try {
            setDetailLoading(true);
            const res = await apiClient.get(`/api/v1/admin/tenants/${tenantId}/transactions`);
            setTransactions(res.data);
        } catch (err) {
            console.error('Error fetching transactions:', err);
        } finally {
            setDetailLoading(false);
        }
    };

    useEffect(() => {
        if (view === 'detail' && selectedTenant) {
            if (activeTab === 'subscription') {
                fetchSubscriptionData(selectedTenant.id);
            } else if (activeTab === 'finance') {
                fetchTransactions(selectedTenant.id);
            }
        }
    }, [activeTab, view, selectedTenant]);

    const handleViewTenant = (tenant) => {
        setSelectedTenant(tenant);
        setView('detail');
        setActiveTab('overview');
    };

    const toggleSelectAll = () => {
        if (selectedIds.length === tenants.length) {
            setSelectedIds([]);
        } else {
            setSelectedIds(tenants.map(t => t.id));
        }
    };

    const toggleSelect = (e, id) => {
        e.stopPropagation();
        setSelectedIds(prev =>
            prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
        );
    };

    const handleBulkAction = async (action) => {
        if (!selectedIds.length) return;
        if (!confirm(t('tenantManager.confirm_bulk', { action: t(`tenantManager.${action}`), count: selectedIds.length }))) return;

        try {
            const url = action === 'delete' 
                ? '/api/v1/admin/tenants/bulk-delete'
                : '/api/v1/admin/tenants/bulk-update';
            
            const body = action === 'delete' 
                ? { ids: selectedIds }
                : { ids: selectedIds, is_active: action === 'activate' };

            const res = await apiClient.post(url, body);

            if (res.status === 200) {
                setSelectedIds([]);
                fetchTenants(page);
            }
        } catch (err) { console.error(err); }
    };
    const handleImpersonate = async (tenantId) => {
        try {
            const res = await apiClient.get(`/api/v1/admin/tenants/${tenantId}/impersonate`);
            const data = res.data;
            const clientUrl = import.meta.env.VITE_CLIENT_WEB_URL || 'http://localhost:5173';
            window.open(`${clientUrl}/login?token=${data.token}`, '_blank');
        } catch (err) {
            console.error('Error impersonating:', err);
            alert('فشل الدخول إلى حساب المستأجر. تحقق من الصلاحيات.');
        }
    };

    const handleCreateTenant = async (e) => {
        e.preventDefault();
        try {
            setCreateLoading(true);
            const res = await apiClient.post('/api/v1/admin/tenants', newTenant);
            if (res.status === 201) {
                setShowAddModal(false);
                setNewTenant({
                    business_name: '',
                    contact_name: '',
                    email: '',
                    phone: '',
                    country_code: 'SA',
                    password: ''
                });
                fetchTenants(1);
                alert('تم إنشاء المستأجر بنجاح');
            }
        } catch (err) {
            console.error('Error creating tenant:', err);
            alert(err.response?.data?.error || 'فشل إنشاء المستأجر');
        } finally {
            setCreateLoading(false);
        }
    };


    const exportToCSV = () => {
        const headers = ['Business Name', 'Contact Name', 'Email', 'Phone', 'Plan', 'Status', 'Balance', 'Devices', 'Joined'];
        const csvContent = [
            headers.join(','),
            ...tenants.map(t => [
                `"${t.business}"`,
                `"${t.name}"`,
                `"${t.email}"`,
                `"${t.phone}"`,
                `"${t.plan}"`,
                `"${t.status}"`,
                t.balance,
                t.devices,
                t.joined
            ].join(','))
        ].join('\n');

        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', `tenants_export_${new Date().toISOString().split('T')[0]}.csv`);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    };

    const handleFilterChange = (e) => {
        const { name, value } = e.target;
        setFilters(prev => ({ ...prev, [name]: value }));
    };

    return (
        <div className="animate-in fade-in zoom-in-95 duration-500">

            {view === 'list' ? (
                <div className="space-y-6">
                    <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                        <div>
                            <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                                <Users className="w-6 h-6 text-rose-500" />
                                {t('tenantManager.title')}
                            </h2>
                            <p className="text-slate-400 text-sm mt-1">{t('tenantManager.description')}</p>
                        </div>
                        <div className="flex flex-wrap items-center gap-3">
                            <div className="relative">
                                <Search className="w-4 h-4 text-slate-500 absolute left-3 top-3.5" />
                                <input
                                    type="text"
                                    placeholder={t('tenantManager.search_placeholder')}
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    className="bg-slate-900 border border-slate-800 rounded-xl pl-10 pr-4 py-2.5 text-sm text-white focus:ring-1 focus:ring-rose-500 outline-none w-64"
                                />
                            </div>
                            <select
                                name="status"
                                value={filters.status}
                                onChange={handleFilterChange}
                                className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-300 focus:ring-1 focus:ring-rose-500 outline-none"
                            >
                                <option value="">{t('tenantManager.all_statuses')}</option>
                                <option value="ACTIVE">{t('tenantManager.active')}</option>
                                <option value="SUSPENDED">{t('tenantManager.suspended')}</option>
                                <option value="PENDING_SETUP">{t('tenantManager.pending_setup')}</option>
                            </select>
                            <select
                                name="sector"
                                value={filters.sector}
                                onChange={handleFilterChange}
                                className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-300 focus:ring-1 focus:ring-rose-500 outline-none"
                            >
                                <option value="">{t('tenantManager.all_sectors')}</option>
                                <option value="Retail">{t('tenantManager.retail')}</option>
                                <option value="Healthcare">{t('tenantManager.healthcare')}</option>
                                <option value="Education">{t('tenantManager.education')}</option>
                                <option value="Finance">{t('tenantManager.finance')}</option>
                                <option value="Technology">{t('tenantManager.technology')}</option>
                                <option value="Other">{t('tenantManager.other')}</option>
                            </select>
                            <button onClick={exportToCSV} className="p-2.5 bg-slate-800 hover:bg-slate-700 rounded-xl text-slate-300 transition-colors" title={t('tenantManager.export_csv')}>
                                <Download className="w-4 h-4" />
                            </button>
                            <button 
                                onClick={() => setShowAddModal(true)}
                                className="flex items-center gap-2 px-4 py-2.5 bg-rose-600 hover:bg-rose-500 text-white rounded-xl text-sm font-bold transition-all shadow-lg shadow-rose-900/20"
                            >
                                <Plus className="w-4 h-4" />
                                {t('tenantManager.add_tenant')}
                            </button>
                        </div>
                    </div>

                    <div className="glass-card overflow-hidden border border-slate-800 bg-slate-900/40 rounded-3xl">
                        <div className="overflow-x-auto w-full">
<table className="w-full text-left">
                            <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                                <tr>
                                    <th className="px-6 py-4 w-10">
                                        <button onClick={toggleSelectAll} className="text-slate-500 hover:text-rose-500 transition-colors">
                                            {selectedIds.length === tenants.length && tenants.length > 0 ? <CheckCircle className="w-5 h-5 text-rose-500" /> : <Shield className="w-5 h-5 opacity-20" />}
                                        </button>
                                    </th>
                                    <th className="px-6 py-4">{t('tenantManager.tenant_info')}</th>
                                    <th className="px-6 py-4">{t('tenantManager.subscription')}</th>
                                    <th className="px-6 py-4">{t('tenantManager.status')}</th>
                                    <th className="px-6 py-4">{t('tenantManager.devices')}</th>
                                    <th className="px-6 py-4">{t('tenantManager.balance')}</th>
                                    <th className="px-6 py-4 text-right">{t('tenantManager.action')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-800 text-slate-300">
                                {loading ? (
                                    <tr>
                                        <td colSpan="6" className="px-6 py-8 text-center text-slate-500">{t('tenantManager.loading')}</td>
                                    </tr>
                                ) : error ? (
                                    <tr>
                                        <td colSpan="6" className="px-6 py-8 text-center text-rose-500">{t('tenantManager.error')} {error}</td>
                                    </tr>
                                ) : tenants.length === 0 ? (
                                    <tr>
                                        <td colSpan="6" className="px-6 py-8 text-center text-slate-500">{t('tenantManager.no_tenants')}</td>
                                    </tr>
                                ) : tenants.map((tenant) => (
                                    <tr key={tenant.id} onClick={() => handleViewTenant(tenant)} className={`hover:bg-slate-800/30 transition-colors cursor-pointer group ${selectedIds.includes(tenant.id) ? 'bg-rose-500/5' : ''}`}>
                                        <td className="px-6 py-4" onClick={(e) => toggleSelect(e, tenant.id)}>
                                            <div className={`w-5 h-5 rounded border flex items-center justify-center transition-all ${selectedIds.includes(tenant.id) ? 'bg-rose-500 border-rose-500' : 'border-slate-700 bg-slate-950'}`}>
                                                {selectedIds.includes(tenant.id) && <CheckCircle className="w-3 h-3 text-white" />}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-3">
                                                <div className="w-10 h-10 rounded-xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-rose-500">
                                                    {tenant.business.charAt(0)}
                                                </div>
                                                <div>
                                                    <div className="font-bold text-white group-hover:text-rose-400 transition-colors">{tenant.business}</div>
                                                    <div className="text-xs text-slate-500">{tenant.name}</div>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`px-2 py-1 rounded-lg text-xs font-bold border ${tenant.plan === 'Platinum Plan' ? 'bg-indigo-500/10 text-indigo-400 border-indigo-500/20' :
                                                tenant.plan === 'Golden Plan' ? 'bg-amber-500/10 text-amber-400 border-amber-500/20' :
                                                    'bg-slate-700/50 text-slate-400 border-slate-700'
                                                }`}>
                                                {tenant.plan}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`flex items-center gap-1.5 text-xs font-bold ${tenant.status === 'Active' ? 'text-emerald-400' : 'text-rose-400'
                                                }`}>
                                                <div className={`w-1.5 h-1.5 rounded-full ${tenant.status === 'Active' ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}`} />
                                                {tenant.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-1 text-xs font-medium text-slate-400">
                                                <Smartphone className="w-4 h-4 text-slate-500" />
                                                {t('tenantManager.linked_devices', { count: tenant.devices })}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 font-mono text-emerald-400 font-bold">
                                            ${tenant.balance.toFixed(2)}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <ChevronRight className="w-5 h-5 text-slate-600 group-hover:text-white inline-block transition-colors" />
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
</div>
                    </div>

                    {selectedIds.length > 0 && (
                        <div className="fixed bottom-8 left-1/2 -translate-x-1/2 bg-slate-900 border border-slate-700 shadow-2xl rounded-2xl p-4 flex items-center gap-6 animate-in slide-in-from-bottom-8 z-50">
                            <div className="text-sm font-bold text-white pl-2">
                                <span className="text-rose-500">{selectedIds.length}</span> {t('tenantManager.tenants_selected')}
                            </div>
                            <div className="h-8 w-px bg-slate-800" />
                            <div className="flex gap-2">
                                <button onClick={() => handleBulkAction('activate')} className="px-4 py-2 bg-emerald-600/10 text-emerald-400 hover:bg-emerald-600 hover:text-white rounded-xl text-xs font-bold transition-all">{t('tenantManager.activate')}</button>
                                <button onClick={() => handleBulkAction('suspend')} className="px-4 py-2 bg-amber-600/10 text-amber-400 hover:bg-amber-600 hover:text-white rounded-xl text-xs font-bold transition-all">{t('tenantManager.suspend')}</button>
                                <button onClick={() => handleBulkAction('delete')} className="px-4 py-2 bg-rose-600/10 text-rose-400 hover:bg-rose-600 hover:text-white rounded-xl text-xs font-bold transition-all flex items-center gap-2">
                                    <Trash2 className="w-3 h-3" /> {t('tenantManager.delete')}
                                </button>
                            </div>
                            <button onClick={() => setSelectedIds([])} className="p-2 text-slate-500 hover:text-white transition-colors">
                                <XCircle className="w-5 h-5" />
                            </button>
                        </div>
                    )}

                    {/* Pagination Controls */}
                    <div className="flex items-center justify-between px-2">
                        <p className="text-xs text-slate-500">
                            {t('tenantManager.showing', {
                                from: (page - 1) * pageSize + 1,
                                to: Math.min(page * pageSize, totalCount),
                                total: totalCount
                            })}
                        </p>
                        <div className="flex gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-xl text-xs font-bold text-white transition-colors"
                            >
                                {t('tenantManager.previous')}
                            </button>
                            <div className="flex items-center gap-1">
                                {[...Array(Math.ceil(totalCount / pageSize))].map((_, i) => (
                                    <button
                                        key={i}
                                        onClick={() => setPage(i + 1)}
                                        className={`w-8 h-8 rounded-lg text-xs font-bold transition-all ${page === i + 1 ? 'bg-rose-500 text-white shadow-lg shadow-rose-900/20' : 'bg-slate-800 text-slate-400 hover:text-white'}`}
                                    >
                                        {i + 1}
                                    </button>
                                ))}
                            </div>
                            <button
                                onClick={() => setPage(p => Math.min(Math.ceil(totalCount / pageSize), p + 1))}
                                disabled={page >= Math.ceil(totalCount / pageSize)}
                                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-xl text-xs font-bold text-white transition-colors"
                            >
                                {t('tenantManager.next')}
                            </button>
                        </div>
                    </div>
                </div>
            ) : (
                <div className="space-y-6">
                    {/* Detail View Header */}
                    <button onClick={() => setView('list')} className="text-slate-400 hover:text-white flex items-center gap-2 text-sm font-bold transition-colors mb-2">
                        <ArrowLeft className="w-4 h-4" /> {t('tenantManager.back_to_list')}
                    </button>

                    <div className="glass-card border border-slate-800 bg-slate-900/40 rounded-3xl p-6 relative overflow-hidden">
                        <div className="absolute top-0 right-0 p-32 bg-rose-600/5 blur-[120px] rounded-full pointer-events-none" />

                        <div className="flex flex-col md:flex-row gap-6 relative z-10">
                            <div className="w-24 h-24 rounded-2xl bg-slate-950 border border-slate-800 flex items-center justify-center text-4xl font-bold text-rose-500 shadow-xl">
                                {selectedTenant.business.charAt(0)}
                            </div>
                            <div className="flex-1">
                                <div className="flex justify-between items-start">
                                    <div>
                                        <h2 className="text-3xl font-bold text-white mb-2">{selectedTenant.business}</h2>
                                        <div className="flex items-center gap-4 text-sm text-slate-400">
                                            <span className="flex items-center gap-1"><Users className="w-4 h-4" /> {selectedTenant.name}</span>
                                            <span className="flex items-center gap-1"><Clock className="w-4 h-4" /> {t('tenantManager.joined')} {selectedTenant.joined}</span>
                                        </div>
                                    </div>
                                    <div className="flex gap-2">
                                        <button
                                            onClick={() => setShowPaymentModal(true)}
                                            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold text-sm transition-colors shadow-lg shadow-emerald-900/20 flex items-center gap-2"
                                        >
                                            <DollarSign className="w-4 h-4" /> {t('tenantManager.manual_top_up')}
                                        </button>
                                        <button 
                                            onClick={() => handleImpersonate(selectedTenant.id)}
                                            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-bold text-sm transition-colors shadow-lg shadow-indigo-900/20 flex items-center gap-2"
                                        >
                                            <Lock className="w-4 h-4" /> {t('tenantManager.login')}
                                        </button>
                                        <button className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold text-sm transition-colors shadow-lg shadow-rose-900/20">{t('tenantManager.suspend_account')}</button>
                                    </div>
                                </div>

                                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-8">
                                    <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/50">
                                        <p className="text-xs text-slate-500 uppercase font-bold">{t('tenantManager.current_balance')}</p>
                                        <p className="text-xl font-bold text-emerald-400 mt-1">${selectedTenant.balance.toFixed(2)}</p>
                                    </div>
                                    <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/50">
                                        <p className="text-xs text-slate-500 uppercase font-bold">{t('tenantManager.plan')}</p>
                                        <p className="text-xl font-bold text-white mt-1">{selectedTenant.plan}</p>
                                    </div>
                                    <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/50">
                                        <p className="text-xs text-slate-500 uppercase font-bold">{t('tenantManager.devices_count')}</p>
                                        <p className="text-xl font-bold text-white mt-1">{selectedTenant.devices} / 10</p>
                                    </div>
                                    <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/50">
                                        <p className="text-xs text-slate-500 uppercase font-bold">{t('tenantManager.status')}</p>
                                        <p className={`text-xl font-bold mt-1 ${selectedTenant.status === 'Active' ? 'text-emerald-400' : 'text-rose-400'}`}>{selectedTenant.status}</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Navigation Tabs */}
                    <div className="flex gap-2 border-b border-slate-800 pb-1">
                        {['overview', 'devices', 'subscription', 'finance'].map(tab => (
                            <button
                                key={tab}
                                onClick={() => setActiveTab(tab)}
                                className={`px-6 py-3 text-sm font-bold capitalize transition-colors relative ${activeTab === tab ? 'text-rose-500' : 'text-slate-500 hover:text-white'
                                    }`}
                            >
                                {t(`tenantManager.tabs.${tab}`)}
                                {activeTab === tab && <div className="absolute bottom-0 left-0 w-full h-0.5 bg-rose-500 rounded-t-full" />}
                            </button>
                        ))}
                    </div>

                    {/* Tab Content */}
                    <div className="min-h-[300px]">
                        {activeTab === 'overview' && (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 animate-in slide-in-from-bottom-2 fade-in">
                                <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6">
                                    <h3 className="font-bold text-white mb-4 flex items-center gap-2">
                                        <Shield className="w-5 h-5 text-rose-500" /> {t('tenantManager.security_access')}
                                    </h3>
                                    <div className="space-y-4">
                                        <div className="flex justify-between items-center p-4 bg-slate-950 rounded-xl border border-slate-800">
                                            <div className="flex items-center gap-3">
                                                <Key className="w-5 h-5 text-slate-500" />
                                                <div>
                                                    <p className="text-sm font-bold text-white">{t('tenantManager.password_reset')}</p>
                                                    <p className="text-xs text-slate-500">{t('tenantManager.password_reset_desc')}</p>
                                                </div>
                                            </div>
                                            <button className="text-xs font-bold text-white bg-slate-800 px-3 py-2 rounded-lg hover:bg-slate-700">{t('tenantManager.send_link')}</button>
                                        </div>
                                        <div className="flex justify-between items-center p-4 bg-slate-950 rounded-xl border border-slate-800">
                                            <div className="flex items-center gap-3">
                                                <Lock className="w-5 h-5 text-slate-500" />
                                                <div>
                                                    <p className="text-sm font-bold text-white">{t('tenantManager.api_access')}</p>
                                                    <p className="text-xs text-slate-500">{t('tenantManager.api_access_desc')}</p>
                                                </div>
                                            </div>
                                            <button className="text-xs font-bold text-white bg-slate-800 px-3 py-2 rounded-lg hover:bg-slate-700">{t('tenantManager.manage_keys')}</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}

                        {activeTab === 'devices' && (
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 animate-in slide-in-from-bottom-2 fade-in">
                                {selectedTenant.device_list.map(device => (
                                    <div key={device.id} className="bg-slate-900 border border-slate-800 rounded-3xl p-6 group hover:border-slate-700 transition-all">
                                        <div className="flex justify-between items-start mb-6">
                                            <div className={`w-12 h-12 rounded-2xl flex items-center justify-center text-xl ${device.status === 'Online' ? 'bg-emerald-500/10 text-emerald-500' : 'bg-slate-800 text-slate-500'
                                                }`}>
                                                <Smartphone className="w-6 h-6" />
                                            </div>
                                            <span className={`px-2 py-1 rounded-lg text-[10px] font-bold uppercase flex items-center gap-1 ${device.status === 'Online' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'
                                                }`}>
                                                {device.status === 'Online' ? <Wifi className="w-3 h-3" /> : <WifiOff className="w-3 h-3" />}
                                                {device.status}
                                            </span>
                                        </div>
                                        <h3 className="text-lg font-bold text-white mb-1">{device.name}</h3>
                                        <p className="text-xs text-slate-500 mb-6 font-mono">ID: {device.id}</p>

                                        <div className="space-y-2 mb-6">
                                            <div className="flex justify-between text-xs">
                                                <span className="text-slate-500">{t('tenantManager.battery')}</span>
                                                <span className={`font-bold flex items-center gap-1 ${device.battery < 20 ? 'text-rose-400' : 'text-emerald-400'}`}>
                                                    <Battery className="w-3 h-3" /> {device.battery}%
                                                </span>
                                            </div>
                                            <div className="flex justify-between text-xs">
                                                <span className="text-slate-500">{t('tenantManager.last_seen')}</span>
                                                <span className="text-slate-400">{device.last_seen}</span>
                                            </div>
                                        </div>

                                        <button className="w-full py-2 rounded-xl text-rose-500 bg-rose-500/10 hover:bg-rose-500 hover:text-white text-xs font-bold flex items-center justify-center gap-2 transition-colors">
                                            <Trash2 className="w-4 h-4" /> {t('tenantManager.unlink_device')}
                                        </button>
                                    </div>
                                ))}
                                {selectedTenant.device_list.length === 0 && (
                                    <div className="col-span-full py-12 text-center text-slate-500 border border-dashed border-slate-800 rounded-3xl">
                                        <Smartphone className="w-12 h-12 mx-auto mb-4 opacity-50" />
                                        <p>{t('tenantManager.no_devices')}</p>
                                    </div>
                                )}
                            </div>
                        )}

                        {activeTab === 'subscription' && (
                            <div className="animate-in slide-in-from-bottom-2 fade-in">
                                {detailLoading ? (
                                    <div className="py-20 text-center text-slate-500">{t('tenantManager.loading_subscription')}</div>
                                ) : subscriptionData ? (
                                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                                        <div className="md:col-span-2 bg-slate-900 border border-slate-800 rounded-3xl p-6">
                                            <h3 className="font-bold text-white mb-6 flex items-center gap-2 text-lg">
                                                <CheckCircle className="w-5 h-5 text-emerald-500" /> {t('tenantManager.active_subscription')}
                                            </h3>
                                            <div className="grid grid-cols-2 gap-8">
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.package_name')}</p>
                                                    <p className="text-xl font-bold text-white">{subscriptionData.package?.name || 'N/A'}</p>
                                                </div>
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.status')}</p>
                                                    <span className="px-2 py-1 bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 rounded-lg text-xs font-bold uppercase">
                                                        {subscriptionData.subscription?.status}
                                                    </span>
                                                </div>
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.start_date')}</p>
                                                    <p className="text-white font-medium">{new Date(subscriptionData.subscription?.start_date).toLocaleDateString()}</p>
                                                </div>
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.expiry_date')}</p>
                                                    <p className="text-white font-medium">{new Date(subscriptionData.subscription?.end_date).toLocaleDateString()}</p>
                                                </div>
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.auto_renew')}</p>
                                                    <p className="text-white font-medium">{subscriptionData.subscription?.auto_renew ? t('tenantManager.enabled') : t('tenantManager.disabled')}</p>
                                                </div>
                                                <div>
                                                    <p className="text-xs text-slate-500 uppercase font-bold mb-1">{t('tenantManager.messages_used')}</p>
                                                    <p className="text-white font-medium">{subscriptionData.subscription?.messages_used} / {subscriptionData.package?.included_messages || 'Unlimited'}</p>
                                                </div>
                                            </div>
                                        </div>
                                        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6">
                                            <h3 className="font-bold text-white mb-4">{t('tenantManager.quick_actions')}</h3>
                                            <div className="space-y-3">
                                                <button className="w-full py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-xl text-sm font-bold transition-colors">{t('tenantManager.change_package')}</button>
                                                <button className="w-full py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-xl text-sm font-bold transition-colors">{t('tenantManager.cancel_subscription')}</button>
                                                <button className="w-full py-3 bg-rose-500/10 hover:bg-rose-500 text-rose-500 hover:text-white rounded-xl text-sm font-bold transition-colors">{t('tenantManager.terminate_immediately')}</button>
                                            </div>
                                        </div>
                                    </div>
                                ) : (
                                    <div className="py-20 text-center text-slate-500">{t('tenantManager.no_active_subscription')}</div>
                                )}
                            </div>
                        )}

                        {activeTab === 'finance' && (
                            <div className="animate-in slide-in-from-bottom-2 fade-in">
                                {detailLoading ? (
                                    <div className="py-20 text-center text-slate-500">{t('tenantManager.loading_transactions')}</div>
                                ) : (
                                    <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden">
                                        <div className="overflow-x-auto w-full">
<table className="w-full text-left font-sans">
                                            <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                                                <tr>
                                                    <th className="px-6 py-4">{t('tenantManager.transaction_id')}</th>
                                                    <th className="px-6 py-4">{t('tenantManager.amount')}</th>
                                                    <th className="px-6 py-4">{t('tenantManager.gateway')}</th>
                                                    <th className="px-6 py-4">{t('tenantManager.status')}</th>
                                                    <th className="px-6 py-4">{t('tenantManager.date')}</th>
                                                </tr>
                                            </thead>
                                            <tbody className="divide-y divide-slate-800 text-slate-300">
                                                {transactions.length === 0 ? (
                                                    <tr>
                                                        <td colSpan="5" className="px-6 py-8 text-center text-slate-500">{t('tenantManager.no_transactions')}</td>
                                                    </tr>
                                                ) : transactions.map(tx => (
                                                    <tr key={tx.id} className="hover:bg-slate-800/30 transition-colors">
                                                        <td className="px-6 py-4 text-xs font-mono">{tx.transaction_id}</td>
                                                        <td className={`px-6 py-4 font-bold ${tx.amount > 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                                                            {tx.amount > 0 ? '+' : ''}${tx.amount.toFixed(2)}
                                                        </td>
                                                        <td className="px-6 py-4 text-xs uppercase text-slate-400">{tx.payment_gateway}</td>
                                                        <td className="px-6 py-4">
                                                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                                                                tx.status === 'completed' ? 'bg-emerald-500/10 text-emerald-400' :
                                                                tx.status === 'pending' ? 'bg-amber-500/10 text-amber-400' :
                                                                'bg-rose-500/10 text-rose-400'
                                                            }`}>
                                                                {tx.status}
                                                            </span>
                                                        </td>
                                                        <td className="px-6 py-4 text-xs text-slate-500">
                                                            {new Date(tx.created_at).toLocaleString()}
                                                        </td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
</div>
                                    </div>
                                )}
                            </div>
                        )}


                    </div>
                </div>
            )}

            {/* Manual Payment Modal */}
            {showPaymentModal && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-md w-full animate-in fade-in zoom-in-95">
                        <div className="flex items-center gap-3 mb-6">
                            <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 flex items-center justify-center">
                                <DollarSign className="w-6 h-6 text-emerald-500" />
                            </div>
                            <div>
                                <h3 className="text-xl font-bold text-white">{t('tenantManager.manual_payment_entry')}</h3>
                                <p className="text-xs text-slate-500">{t('tenantManager.add_balance_for', { business: selectedTenant?.business })}</p>
                            </div>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.amount_sar')}</label>
                                <div className="relative">
                                    <DollarSign className="w-4 h-4 text-slate-500 absolute left-3 top-3.5" />
                                    <input
                                        type="number"
                                        placeholder="0.00"
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-10 pr-4 py-3 text-white focus:outline-none focus:border-emerald-500"
                                    />
                                </div>
                            </div>

                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.payment_method')}</label>
                                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-emerald-500">
                                    <option>{t('tenantManager.bank_transfer')}</option>
                                    <option>{t('tenantManager.cash')}</option>
                                    <option>{t('tenantManager.stc_pay')}</option>
                                    <option>{t('tenantManager.credit_card')}</option>
                                    <option>{t('tenantManager.other')}</option>
                                </select>
                            </div>

                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.reference_optional')}</label>
                                <input
                                    type="text"
                                    placeholder={t('tenantManager.reference_placeholder')}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-emerald-500"
                                />
                            </div>

                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.notes')}</label>
                                <textarea
                                    rows="3"
                                    placeholder={t('tenantManager.notes_placeholder')}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-emerald-500 resize-none"
                                />
                            </div>
                        </div>

                        <div className="flex gap-3 mt-8">
                            <button
                                onClick={() => setShowPaymentModal(false)}
                                className="flex-1 py-3 text-slate-400 font-bold hover:text-white transition-colors"
                            >
                                {t('tenantManager.cancel')}
                            </button>
                            <button className="flex-1 py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold transition-colors shadow-lg shadow-emerald-900/20">
                                {t('tenantManager.add_payment')}
                            </button>
                        </div>
                    </div>
                </div>
            )}
            
            {/* Add Tenant Modal */}
            {showAddModal && (
                <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-300">
                    <div className="bg-slate-900 border border-slate-800 w-full max-w-lg rounded-3xl p-8 shadow-2xl animate-in zoom-in-95 duration-300 overflow-hidden relative">
                        <div className="flex justify-between items-center mb-8">
                            <div>
                                <h3 className="text-xl font-bold text-white">{t('tenantManager.add_new_tenant')}</h3>
                                <p className="text-sm text-slate-500 mt-1">{t('tenantManager.add_tenant_desc')}</p>
                            </div>
                            <button 
                                onClick={() => setShowAddModal(false)}
                                className="p-2 hover:bg-slate-800 rounded-xl text-slate-500 transition-colors"
                            >
                                <XCircle className="w-6 h-6" />
                            </button>
                        </div>

                        <form onSubmit={handleCreateTenant} className="space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="col-span-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.business_name')}</label>
                                    <input
                                        type="text"
                                        required
                                        value={newTenant.business_name}
                                        onChange={(e) => setNewTenant({...newTenant, business_name: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder={t('tenantManager.business_placeholder')}
                                    />
                                </div>
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.contact_person')}</label>
                                    <input
                                        type="text"
                                        required
                                        value={newTenant.contact_name}
                                        onChange={(e) => setNewTenant({...newTenant, contact_name: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder={t('tenantManager.full_name')}
                                    />
                                </div>
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.email_address')}</label>
                                    <input
                                        type="email"
                                        required
                                        value={newTenant.email}
                                        onChange={(e) => setNewTenant({...newTenant, email: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder={t('tenantManager.email_placeholder')}
                                    />
                                </div>
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.phone_number')}</label>
                                    <input
                                        type="text"
                                        required
                                        value={newTenant.phone}
                                        onChange={(e) => setNewTenant({...newTenant, phone: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder={t('tenantManager.phone_placeholder')}
                                    />
                                </div>
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('tenantManager.initial_password')}</label>
                                    <input
                                        type="password"
                                        required
                                        minLength="8"
                                        value={newTenant.password}
                                        onChange={(e) => setNewTenant({...newTenant, password: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder={t('tenantManager.password_placeholder')}
                                    />
                                </div>
                            </div>

                            <div className="flex gap-3 mt-8 pt-4 border-t border-slate-800">
                                <button
                                    type="button"
                                    onClick={() => setShowAddModal(false)}
                                    className="flex-1 py-3 text-slate-400 font-bold hover:text-white transition-colors"
                                >
                                    {t('tenantManager.cancel')}
                                </button>
                                <button 
                                    type="submit"
                                    disabled={createLoading}
                                    className="flex-1 py-3 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold transition-all shadow-lg shadow-rose-900/20 disabled:opacity-50 flex items-center justify-center gap-2"
                                >
                                    {createLoading ? t('tenantManager.creating') : t('tenantManager.create_tenant')}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

        </div>
    );
}
