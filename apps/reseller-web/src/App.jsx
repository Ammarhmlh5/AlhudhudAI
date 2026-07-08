import React, { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { 
  Users, Building2, LayoutDashboard, Key, 
  Settings, LogOut, Menu, X, Plus, 
  Search, ShieldCheck, UserPlus, Globe
} from 'lucide-react';
import apiClient from './api/axios';
import Login from './components/Login';

const ProtectedRoute = ({ children }) => {
    const token = localStorage.getItem('reseller_token');
    if (!token) return <Navigate to="/login" replace />;
    return children;
};

// ── Components ────────────────────────────────────────────────────────────────
const SidebarItem = ({ icon: Icon, label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center gap-4 px-6 py-4 transition-all duration-300 ${
      active
        ? 'text-rose-500 bg-rose-500/5 border-r-4 border-rose-500'
        : 'text-slate-400 hover:text-slate-100 hover:bg-slate-900/50'
    }`}
  >
    <Icon className={`w-5 h-5 ${active ? 'text-rose-500' : 'text-slate-500'}`} />
    <span className="font-bold text-sm tracking-wide">{label}</span>
  </button>
);

const Dashboard = () => {
    const [stats, setStats] = useState({ tenants: 0, staff: 0, messages: 0 });
    
    useEffect(() => {
        // Fetch stats from backend
        setStats({ tenants: 12, staff: 4, messages: 125430 });
    }, []);

    return (
        <div className="space-y-8 animate-fade-in">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-4xl font-black text-white tracking-tighter">B2B Reseller Portal</h1>
                    <p className="text-slate-500 font-bold uppercase tracking-widest text-xs mt-2">Manage your software customers and team</p>
                </div>
                <button className="premium-button flex items-center gap-2">
                    <Plus className="w-4 h-4" /> Add New Customer
                </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {[
                    { label: 'Total Tenants', value: stats.tenants, icon: Building2, color: 'text-blue-500' },
                    { label: 'Active Staff', value: stats.staff, icon: Users, color: 'text-emerald-500' },
                    { label: 'Messages Routed', value: stats.messages.toLocaleString(), icon: Globe, color: 'text-rose-500' },
                ].map((stat, i) => (
                    <div key={i} className="glass-card p-8 group hover:border-slate-700 transition-all">
                        <div className="flex justify-between items-start mb-4">
                            <div className={`p-4 rounded-2xl bg-slate-950 border border-slate-800 ${stat.color}`}>
                                <stat.icon className="w-6 h-6" />
                            </div>
                            <span className="text-[10px] font-black text-slate-600 uppercase tracking-widest">Lifetime</span>
                        </div>
                        <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mb-1">{stat.label}</p>
                        <h3 className="text-4xl font-black text-white">{stat.value}</h3>
                    </div>
                ))}
            </div>
        </div>
    );
};

const TenantsList = () => {
    const [tenants, setTenants] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        apiClient.get('/api/v1/reseller/tenants')
            .then(res => setTenants(res.data))
            .catch(err => console.error(err))
            .finally(() => setLoading(false));
    }, []);

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex justify-between items-end">
                <div>
                    <h2 className="text-3xl font-black text-white tracking-tight">Software Customers</h2>
                    <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mt-1">Directly managed tenants using your API keys</p>
                </div>
                <div className="relative">
                    <Search className="w-4 h-4 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                    <input 
                        type="text" 
                        placeholder="Search customers..." 
                        className="bg-slate-900 border border-slate-800 rounded-2xl pl-12 pr-6 py-3 text-sm text-white focus:outline-none focus:border-rose-500 w-64 transition-all"
                    />
                </div>
            </div>

            <div className="glass-card overflow-hidden">
                <table className="w-full text-left">
                    <thead>
                        <tr className="bg-slate-950/50 border-b border-slate-800">
                            <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">Business Name</th>
                            <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">Contact</th>
                            <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">Status</th>
                            <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">API Key</th>
                            <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800">
                        {loading ? (
                            <tr><td colSpan="5" className="px-8 py-20 text-center text-slate-500 font-bold italic">Loading your customers...</td></tr>
                        ) : tenants.length === 0 ? (
                            <tr><td colSpan="5" className="px-8 py-20 text-center text-slate-500 font-bold italic">No customers found. Start by adding your first one.</td></tr>
                        ) : tenants.map(t => (
                            <tr key={t.id} className="hover:bg-slate-900/30 transition-all group">
                                <td className="px-8 py-6">
                                    <div className="font-bold text-white group-hover:text-rose-500 transition-colors">{t.business_name}</div>
                                    <div className="text-[10px] text-slate-500 font-mono mt-1">{t.id}</div>
                                </td>
                                <td className="px-8 py-6">
                                    <div className="text-sm text-slate-300">{t.contact_name}</div>
                                    <div className="text-xs text-slate-500">{t.email}</div>
                                </td>
                                <td className="px-8 py-6">
                                    <span className={`px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest ${t.is_active ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}`}>
                                        {t.is_active ? 'Active' : 'Suspended'}
                                    </span>
                                </td>
                                <td className="px-8 py-6">
                                    <div className="flex items-center gap-2">
                                        <code className="text-xs text-slate-400 bg-slate-950 px-2 py-1 rounded border border-slate-800">
                                            {t.api_key_accounting ? '••••••••' : 'Not Set'}
                                        </code>
                                        <button className="p-2 hover:bg-slate-800 rounded-lg text-slate-500 transition-all">
                                            <Key className="w-3 h-3" />
                                        </button>
                                    </div>
                                </td>
                                <td className="px-8 py-6 text-right">
                                    <button className="text-slate-500 hover:text-white font-bold text-xs uppercase tracking-widest transition-all">Settings</button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

const StaffManager = () => {
    const [staff, setStaff] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        apiClient.get('/api/v1/reseller/staff')
            .then(res => setStaff(res.data))
            .catch(err => console.error(err))
            .finally(() => setLoading(false));
    }, []);

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex justify-between items-end">
                <div>
                    <h2 className="text-3xl font-black text-white tracking-tight">Your Team</h2>
                    <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mt-1">Management staff for your software company</p>
                </div>
                <button className="premium-button flex items-center gap-2">
                    <UserPlus className="w-4 h-4" /> Invite Staff
                </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {loading ? (
                    <div className="col-span-full py-20 text-center text-slate-500">Loading team...</div>
                ) : staff.length === 0 ? (
                    <div className="col-span-full py-20 text-center text-slate-500 glass-card">No staff members yet.</div>
                ) : staff.map(s => (
                    <div key={s.id} className="glass-card p-6 flex items-center gap-4 group">
                        <div className="w-12 h-12 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-500 font-black text-xl">
                            {s.full_name.charAt(0)}
                        </div>
                        <div className="flex-1">
                            <h4 className="text-white font-bold group-hover:text-rose-500 transition-all">{s.full_name}</h4>
                            <p className="text-[10px] text-slate-500 font-black uppercase tracking-widest">{s.role?.name || 'Staff'}</p>
                        </div>
                        <button className="p-2 text-slate-600 hover:text-white transition-all">
                            <Settings className="w-4 h-4" />
                        </button>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ── Main Layout ──────────────────────────────────────────────────────────────
const MainApp = () => {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const navigate = useNavigate();

  const handleTabClick = (tabId) => {
    setActiveTab(tabId);
    setIsMobileMenuOpen(false);
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard': return <Dashboard />;
      case 'tenants': return <TenantsList />;
      case 'staff': return <StaffManager />;
      default: return <Dashboard />;
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex font-sans w-full selection:bg-rose-500 selection:text-white">
      {/* Sidebar */}
      <aside className={`w-72 bg-slate-950 border-r border-slate-900 flex flex-col fixed inset-y-0 start-0 z-50 transition-transform duration-300 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        <div className="h-24 flex items-center justify-between px-8 border-b border-slate-900">
          <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-rose-600 rounded-2xl flex items-center justify-center shadow-lg shadow-rose-900/40">
                  <ShieldCheck className="w-6 h-6 text-white" />
              </div>
              <span className="text-xl font-black text-white tracking-tighter uppercase">Hudhud<span className="text-rose-500">B2B</span></span>
          </div>
          <button className="md:hidden text-slate-400 hover:text-white" onClick={() => setIsMobileMenuOpen(false)}>
            <X className="w-6 h-6" />
          </button>
        </div>

        <nav className="flex-1 py-8">
          <SidebarItem icon={LayoutDashboard} label="Overview" active={activeTab === 'dashboard'} onClick={() => handleTabClick('dashboard')} />
          <SidebarItem icon={Building2} label="Customers" active={activeTab === 'tenants'} onClick={() => handleTabClick('tenants')} />
          <SidebarItem icon={Users} label="Our Team" active={activeTab === 'staff'} onClick={() => handleTabClick('staff')} />
          
          <div className="px-8 mt-8 mb-4">
              <div className="h-px bg-slate-900 w-full" />
              <p className="text-[10px] font-black text-slate-600 uppercase tracking-[0.2em] mt-4">Integrations</p>
          </div>
          <SidebarItem icon={Key} label="API Settings" active={activeTab === 'api'} onClick={() => handleTabClick('api')} />
          <SidebarItem icon={Settings} label="Portal Config" active={activeTab === 'settings'} onClick={() => handleTabClick('settings')} />
        </nav>

        <div className="p-6 border-t border-slate-900">
          <button 
            onClick={() => {
                localStorage.removeItem('reseller_token');
                navigate('/login');
            }}
            className="w-full flex items-center gap-4 px-6 py-4 text-slate-500 hover:text-rose-500 transition-all rounded-2xl hover:bg-rose-500/5 group"
          >
            <LogOut className="w-5 h-5 group-hover:scale-110 transition-transform" />
            <span className="font-bold text-sm">Sign Out</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 md:ms-72 p-12 w-full">
        {renderContent()}
      </main>
    </div>
  );
}

export default function App() {
    return (
        <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/*" element={
                <ProtectedRoute>
                    <MainApp />
                </ProtectedRoute>
            } />
        </Routes>
    );
}
