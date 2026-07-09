import { useState, useEffect } from 'react';
import { Users, UserPlus, Shield, Activity, Lock, Eye, Edit, Trash, CheckSquare, Search, FileText, Plus, Download, XCircle } from 'lucide-react';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

export default function StaffManager() {
    const [activeTab, setActiveTab] = useState('list');
    const [showModal, setShowModal] = useState(false);

    const [users, setUsers] = useState([]);
    const [roles, setRoles] = useState([]);
    const [loading, setLoading] = useState(true);
    const [selectedIds, setSelectedIds] = useState([]);

    // Filtering state
    const [searchQuery, setSearchQuery] = useState('');
    const [debouncedSearch, setDebouncedSearch] = useState('');
    const [filters, setFilters] = useState({
        role_id: '',
        department: ''
    });

    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(searchQuery);
        }, 500);
        return () => clearTimeout(timer);
    }, [searchQuery]);

    useEffect(() => {
        fetchStaff();
    }, [debouncedSearch, filters]);

    useEffect(() => {
        fetchRoles();
    }, []);

    const fetchStaff = async () => {
        try {
            setLoading(true);
            const params = {};
            if (debouncedSearch) params.search = debouncedSearch;
            if (filters.role_id) params.role_id = filters.role_id;
            if (filters.department) params.department = filters.department;
            
            const res = await apiClient.get('/api/v1/staff/', { params });
            if (res.status === 200) {
                setUsers(res.data);
            }
        } catch (err) { console.error(err); }
        finally { setLoading(false); }
    };

    const fetchRoles = async () => {
        try {
            const res = await apiClient.get('/api/v1/staff/roles');
            if (res.status === 200) {
                setRoles(res.data);
            }
        } catch (err) { console.error(err); }
    };

    const handleBulkAction = async (action) => {
        if (!selectedIds.length) return;
        if (!confirm(`Are you sure you want to ${action} ${selectedIds.length} staff members?`)) return;

        try {
            const url = action === 'delete' 
                ? '/api/v1/staff/bulk-delete'
                : '/api/v1/staff/bulk-update';
            
            const body = { ids: selectedIds };
            if (action === 'deactivate') {
                body.is_active = false;
            } else if (action === 'activate') {
                body.is_active = true;
            }

            const res = await apiClient.post(url, body);

            if (res.status === 200) {
                setSelectedIds([]);
                fetchStaff();
            }
        } catch (err) { console.error(err); }
    };

    const toggleSelect = (e, id) => {
        e.stopPropagation();
        setSelectedIds(prev => prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]);
    };

    const allPermissions = [
        { module: 'Dashboard', actions: ['view_stats'] },
        { module: 'Tenants', actions: ['view_list', 'edit_tenant', 'suspend_tenant'] },
        { module: 'Finance', actions: ['view_income', 'approve_payments', 'refund_payments'] },
        { module: 'Support', actions: ['view_tickets', 'reply_tickets', 'delete_tickets'] },
        { module: 'System', actions: ['manage_staff', 'manage_gateways', 'view_logs'] },
    ];

    const exportToCSV = () => {
        const headers = ['Name', 'Email', 'Role', 'Status', 'Department'];
        const csvContent = [
            headers.join(','),
            ...users.map(u => [
                `"${u.full_name || u.name || ''}"`,
                `"${u.user?.email || u.email || ''}"`,
                `"${u.role?.name || u.role || ''}"`,
                `"${u.is_active || u.status === 'Active' ? 'Active' : 'Deactivated'}"`,
                `"${u.department || u.last_login || ''}"`
            ].join(','))
        ].join('\n');

        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', `staff_export_${new Date().toISOString().split('T')[0]}.csv`);
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
        <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">

            {/* Header & Tabs */}
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
                <div>
                    <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                        <Users className="w-6 h-6 text-rose-500" />
                        Staff & Permissions
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">Manage employees, define access roles, and audit activities.</p>
                </div>
                <div className="flex bg-slate-900 p-1 rounded-xl border border-slate-800">
                    <button onClick={() => setActiveTab('list')} className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'list' ? 'bg-slate-800 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}>Staff List</button>
                    <button onClick={() => setActiveTab('roles')} className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'roles' ? 'bg-slate-800 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}>Roles & Access</button>
                    <button onClick={() => setActiveTab('logs')} className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'logs' ? 'bg-slate-800 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}>Activity Reports</button>
                </div>
            </div>

            {activeTab === 'list' && (
                <div className="glass-card overflow-hidden border border-slate-800 bg-slate-900/40 rounded-3xl">
                    <div className="p-6 border-b border-slate-800 flex flex-col xl:flex-row justify-between items-start xl:items-center gap-4">
                        <div className="flex flex-wrap items-center gap-3">
                            <div className="relative">
                                <Search className="w-4 h-4 text-slate-500 absolute left-3 top-3.5" />
                                <input
                                    type="text"
                                    placeholder="Search staff..."
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    className="bg-slate-900 border border-slate-800 rounded-xl pl-10 pr-4 py-2.5 text-sm text-white focus:ring-1 focus:ring-rose-500 outline-none w-64"
                                />
                            </div>
                            <select
                                name="role_id"
                                value={filters.role_id}
                                onChange={handleFilterChange}
                                className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-300 focus:ring-1 focus:ring-rose-500 outline-none w-40"
                            >
                                <option value="">All Roles</option>
                                {roles.map(r => (
                                    <option key={r.id} value={r.id}>{r.name}</option>
                                ))}
                            </select>
                            <select
                                name="department"
                                value={filters.department}
                                onChange={handleFilterChange}
                                className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-300 focus:ring-1 focus:ring-rose-500 outline-none w-40"
                            >
                                <option value="">All Departments</option>
                                <option value="Management">Management</option>
                                <option value="IT">IT</option>
                                <option value="Support">Support</option>
                                <option value="Sales">Sales</option>
                                <option value="Finance">Finance</option>
                                <option value="Marketing">Marketing</option>
                                <option value="Customer Success">Customer Success</option>
                            </select>
                            <button onClick={exportToCSV} className="p-2.5 bg-slate-800 hover:bg-slate-700 rounded-xl text-slate-300 transition-colors" title="Export to CSV">
                                <Download className="w-4 h-4" />
                            </button>
                        </div>
                        <button onClick={() => setShowModal(true)} className="bg-rose-600 hover:bg-rose-500 text-white px-4 py-2.5 rounded-xl text-sm font-bold flex items-center gap-2 transition-colors shrink-0">
                            <UserPlus className="w-4 h-4" /> Add New Staff
                        </button>
                    </div>
                    <div className="overflow-x-auto w-full">
<table className="w-full text-left">
                        <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                            <tr>
                                <th className="px-6 py-4 w-10">
                                    <CheckSquare className="w-4 h-4 opacity-20" />
                                </th>
                                <th className="px-6 py-4">Employee</th>
                                <th className="px-6 py-4">Role</th>
                                <th className="px-6 py-4">Status</th>
                                <th className="px-6 py-4">Department</th>
                                <th className="px-6 py-4 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800 text-slate-300">
                            {users.map((user) => (
                                 <tr key={user.id} className={`hover:bg-slate-800/30 transition-colors ${selectedIds.includes(user.id) ? 'bg-rose-500/5' : ''}`}>
                                    <td className="px-6 py-4" onClick={(e) => toggleSelect(e, user.id)}>
                                        <div className={`w-5 h-5 rounded border flex items-center justify-center transition-all ${selectedIds.includes(user.id) ? 'bg-rose-500 border-rose-500' : 'border-slate-700 bg-slate-950'}`}>
                                            {selectedIds.includes(user.id) && <CheckSquare className="w-3 h-3 text-white" />}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-3">
                                             <div className="w-10 h-10 rounded-full bg-slate-800 flex items-center justify-center font-bold text-rose-500 border border-slate-700">
                                                {(user.full_name || user.name || 'U').charAt(0)}
                                            </div>
                                            <div>
                                                <div className="font-bold text-white">{user.full_name || user.name}</div>
                                                <div className="text-xs text-slate-500">{user.user?.email || user.email || 'N/A'}</div>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4"><span className="bg-slate-800 text-white px-2 py-1 rounded-lg text-xs font-bold border border-slate-700">{user.role?.name || user.role}</span></td>
                                    <td className="px-6 py-4">
                                        <span className={`text-xs font-bold flex items-center gap-1 ${user.is_active || user.status === 'Active' ? 'text-emerald-400' : 'text-slate-500'}`}>
                                            <span className={`w-2 h-2 rounded-full ${user.is_active || user.status === 'Active' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-600'}`} /> 
                                            {user.is_active || user.status === 'Active' ? 'Active' : 'Deactivated'}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 text-xs font-mono text-slate-500">{user.department || user.last_login || '-'}</td>
                                    <td className="px-6 py-4 text-right">
                                        <button className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"><Edit className="w-4 h-4" /></button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
</div>

                    {selectedIds.length > 0 && (
                        <div className="fixed bottom-8 left-1/2 -translate-x-1/2 bg-slate-900 border border-slate-700 shadow-2xl rounded-2xl p-4 flex items-center gap-6 animate-in slide-in-from-bottom-8 z-50">
                            <div className="text-sm font-bold text-white pl-2">
                                <span className="text-rose-500">{selectedIds.length}</span> Staff Selected
                            </div>
                            <div className="h-8 w-px bg-slate-800" />
                            <div className="flex gap-2">
                                <button onClick={() => handleBulkAction('activate')} className="px-4 py-2 bg-emerald-600/10 text-emerald-400 hover:bg-emerald-600 hover:text-white rounded-xl text-xs font-bold transition-all">Activate</button>
                                <button onClick={() => handleBulkAction('deactivate')} className="px-4 py-2 bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white rounded-xl text-xs font-bold transition-all">Deactivate</button>
                                <button onClick={() => handleBulkAction('delete')} className="px-4 py-2 bg-rose-600/10 text-rose-400 hover:bg-rose-600 hover:text-white rounded-xl text-xs font-bold transition-all flex items-center gap-2">
                                    <Trash className="w-3 h-3" /> Delete
                                </button>
                            </div>
                            <button onClick={() => setSelectedIds([])} className="p-2 text-slate-500 hover:text-white transition-colors">
                                <XCircle className="w-5 h-5 text-rose-500" />
                            </button>
                        </div>
                    )}
                </div>
            )}

            {activeTab === 'roles' && (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Roles List */}
                    <div className="lg:col-span-1 space-y-4">
                        {roles.map(role => (
                            <div key={role.id} className="bg-slate-900 border border-slate-800 p-4 rounded-2xl cursor-pointer hover:border-rose-500/50 transition-colors group">
                                <div className="flex justify-between items-center">
                                    <h4 className="font-bold text-white">{role.name}</h4>
                                    <Shield className="w-4 h-4 text-slate-500 group-hover:text-rose-500 transition-colors" />
                                </div>
                                <p className="text-xs text-slate-500 mt-2">{role.permissions.includes('all') ? 'Full System Access' : `${role.permissions.length} Limited Permissions`}</p>
                            </div>
                        ))}
                        <button className="w-full py-3 border border-dashed border-slate-700 rounded-2xl text-slate-400 hover:text-white hover:bg-slate-800 transition-colors text-sm font-bold flex items-center justify-center gap-2">
                            <Plus className="w-4 h-4" /> Create New Role
                        </button>
                    </div>

                    {/* Permissions Matrix */}
                    <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-3xl p-6">
                        <h3 className="font-bold text-white mb-6 flex items-center gap-2">
                            <Lock className="w-5 h-5 text-rose-500" />
                            Access Control Matrix
                        </h3>
                        <div className="space-y-6">
                            {allPermissions.map((mod, idx) => (
                                <div key={idx} className="border-b border-slate-800/50 pb-4 last:border-0 last:pb-0">
                                    <h4 className="text-sm font-bold text-slate-300 mb-3 uppercase tracking-wider">{mod.module}</h4>
                                    <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                                        {mod.actions.map(action => (
                                            <label key={action} className="flex items-center gap-2 cursor-pointer group">
                                                <div className="w-4 h-4 rounded border border-slate-600 bg-slate-800 group-hover:border-rose-500 transition-colors flex items-center justify-center">
                                                    <div className="w-2.5 h-2.5 bg-rose-500 rounded-sm opacity-0 group-hover:opacity-100" />
                                                </div>
                                                <span className="text-xs text-slate-400 group-hover:text-slate-200">{action.replace('_', ' ')}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            )}

            {activeTab === 'logs' && (
                <div className="glass-card border border-slate-800 bg-slate-900/40 rounded-3xl overflow-hidden p-6">
                    <div className="flex justify-between items-center mb-6">
                        <h3 className="text-lg font-bold text-white flex items-center gap-2">
                            <Activity className="w-5 h-5 text-rose-500" />
                            Audit Log & Activity
                        </h3>
                        <div className="flex gap-2">
                            <input type="date" className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-1 text-xs text-white" />
                            <button className="p-2 bg-slate-800 rounded-lg text-slate-400 hover:text-white"><Download className="w-4 h-4" /></button>
                        </div>
                    </div>

                    <div className="space-y-4">
                        {logs.map(log => (
                            <div key={log.id} className="flex items-center justify-between p-4 bg-slate-950/50 rounded-2xl border border-slate-800/50 hover:bg-slate-900 transition-colors">
                                <div className="flex items-center gap-4">
                                    <div className="w-2 h-2 rounded-full bg-rose-500" />
                                    <div>
                                        <div className="text-sm font-bold text-white">
                                            <span className="text-rose-400">{log.user}</span> {log.action} <span className="text-emerald-400">[{log.target}]</span>
                                        </div>
                                        <div className="text-xs text-slate-500 mt-0.5 font-mono">{log.time}</div>
                                    </div>
                                </div>
                                <button className="text-xs text-slate-500 hover:text-white underline">Details</button>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {showModal && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-md w-full animate-in fade-in zoom-in-95">
                        <h3 className="text-xl font-bold text-white mb-6">Add New Staff Member</h3>
                        <div className="space-y-4">
                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">Full Name</label>
                                <input type="text" className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500" />
                            </div>
                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">Email Address</label>
                                <input type="email" className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500" />
                            </div>
                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">Assign Role</label>
                                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500">
                                    <option>Support Agent</option>
                                    <option>Accountant</option>
                                    <option>Tech Admin</option>
                                </select>
                            </div>
                        </div>
                        <div className="flex gap-3 mt-8">
                            <button onClick={() => setShowModal(false)} className="flex-1 py-3 text-slate-400 font-bold hover:text-white transition-colors">Cancel</button>
                            <button className="flex-1 py-3 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold transition-colors shadow-lg shadow-rose-900/20">Create Account</button>
                        </div>
                    </div>
                </div>
            )}

        </div>
    );
}
