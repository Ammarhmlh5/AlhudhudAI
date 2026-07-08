import React, { useState, useEffect } from 'react';
import { 
  Server, Plus, Search, Filter, AlertTriangle, 
  CheckCircle, XCircle, Clock, TrendingUp, Activity,
  Database, Shield, Globe, MoreVertical, X, ChevronRight
} from 'lucide-react';
import GatewayDetails from './GatewayDetails';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

export default function Gateways() {
  const [gateways, setGateways] = useState([]);
  const [filteredGateways, setFilteredGateways] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [selectedGateway, setSelectedGateway] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createForm, setCreateForm] = useState({
    tenant_id: '',
    gateway_name: '',
    subscription_type: 'monthly',
    subscription_fee: 0,
    server_rental_fee: 0
  });

  useEffect(() => {
    fetchGateways();
    fetchAlerts();
  }, []);

  useEffect(() => {
    filterGateways();
  }, [gateways, searchTerm, statusFilter]);

  const fetchGateways = async () => {
    try {
      setLoading(true);
      const res = await apiClient.get('/api/v1/admin/gateways');
      setGateways(res.data || []);
    } catch (err) {
      console.error('Failed to fetch gateways:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchAlerts = async () => {
    try {
      const res = await apiClient.get('/api/v1/admin/gateways/alerts');
      setAlerts(res.data || []);
    } catch (err) {
      console.error('Failed to fetch alerts:', err);
    }
  };

  const filterGateways = () => {
    let filtered = gateways;
    if (searchTerm) {
      filtered = filtered.filter(g => 
        g.gateway_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        g.tenant_name?.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }
    if (statusFilter !== 'all') {
      filtered = filtered.filter(g => g.status === statusFilter);
    }
    setFilteredGateways(filtered);
  };

  const handleCreateGateway = async (e) => {
    e.preventDefault();
    try {
      const res = await apiClient.post('/api/v1/admin/gateways', createForm);
      if (res.status === 200 || res.status === 201) {
        setShowCreateForm(false);
        setCreateForm({
          tenant_id: '',
          gateway_name: '',
          subscription_type: 'monthly',
          subscription_fee: 0,
          server_rental_fee: 0
        });
        fetchGateways();
      }
    } catch (err) {
      console.error('Failed to create gateway:', err);
    }
  };

  if (selectedGateway) {
    return (
      <div className="p-8 animate-in slide-in-from-right duration-500">
        <button
          onClick={() => setSelectedGateway(null)}
          className="mb-8 px-6 py-2 bg-slate-900 text-slate-400 border border-slate-800 rounded-xl hover:text-white hover:bg-slate-800 transition-all flex items-center gap-2 font-bold uppercase tracking-widest text-xs"
        >
          ← Back to Network
        </button>
        <GatewayDetails gatewayId={selectedGateway} onClose={() => setSelectedGateway(null)} />
      </div>
    );
  }

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-rose-600 rounded-2xl shadow-lg shadow-rose-900/20">
              <Database className="w-8 h-8 text-white" />
            </div>
            Dedicated Gateways
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">
            Enterprise grade secure messaging infrastructure
          </p>
        </div>
        <button
          onClick={() => setShowCreateForm(true)}
          className="px-8 py-4 bg-rose-600 text-white rounded-2xl hover:bg-rose-500 shadow-xl shadow-rose-900/20 transition-all flex items-center gap-3 font-black uppercase tracking-widest text-sm"
        >
          <Plus className="w-5 h-5" />
          Deploy New Gateway
        </button>
      </div>

      {/* Active Alerts */}
      {alerts.length > 0 && (
        <div className="bg-amber-500/10 border border-amber-500/20 p-6 rounded-[2rem] flex items-start gap-4">
          <div className="p-3 bg-amber-500/20 rounded-2xl">
            <AlertTriangle className="w-6 h-6 text-amber-500" />
          </div>
          <div className="flex-1">
            <h3 className="font-black text-amber-400 uppercase tracking-widest text-sm mb-2">Network Alerts ({alerts.length})</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {alerts.slice(0, 3).map((alert, idx) => (
                <div key={idx} className="text-xs text-slate-400 font-medium flex items-center gap-2">
                  <div className="w-1.5 h-1.5 bg-amber-500 rounded-full" />
                  <span className="font-bold text-slate-300">{alert.gateway_name}:</span> {alert.message}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <StatCard icon={Database} label="Total Assets" value={gateways.length} color="rose" />
        <StatCard icon={CheckCircle} label="Active" value={gateways.filter(g => g.status === 'active').length} color="emerald" />
        <StatCard icon={Clock} label="Suspended" value={gateways.filter(g => g.status === 'suspended').length} color="amber" />
        <StatCard icon={XCircle} label="Terminated" value={gateways.filter(g => g.status === 'terminated').length} color="slate" />
      </div>

      {/* Filters Bar */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        <div className="flex flex-col md:flex-row gap-6">
          <div className="flex-1 relative group">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-rose-500 transition-colors" />
            <input
              type="text"
              placeholder="Search by gateway name or tenant..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-12 pr-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white placeholder:text-slate-600 focus:outline-none focus:border-rose-500 transition-all shadow-inner"
            />
          </div>
          <div className="flex items-center gap-4">
            <div className="relative">
                <Filter className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="pl-12 pr-8 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-sm text-slate-300 focus:outline-none focus:border-rose-500 appearance-none cursor-pointer min-w-[180px]"
                >
                <option value="all">All Status</option>
                <option value="active">Active</option>
                <option value="suspended">Suspended</option>
                <option value="terminated">Terminated</option>
                </select>
            </div>
          </div>
        </div>
      </div>

      {/* Gateways Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-32 space-y-4">
            <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
            <p className="text-slate-400 font-bold animate-pulse">Scanning Network Infrastructure...</p>
          </div>
        ) : filteredGateways.length === 0 ? (
          <div className="text-center py-24">
            <Globe className="w-16 h-16 text-slate-800 mx-auto mb-4" />
            <p className="text-slate-500 font-bold text-lg">No dedicated gateways deployed.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="bg-slate-950/50 border-b border-slate-800">
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">Gateway Asset</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">Owner / Tenant</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">Live Status</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">Plan Details</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">Performance</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {filteredGateways.map((gateway) => (
                  <tr key={gateway.id} className="hover:bg-slate-800/30 transition-all group">
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 rounded-2xl bg-slate-950 border border-slate-800 flex items-center justify-center group-hover:border-rose-500/30 transition-colors">
                          <Database className="w-6 h-6 text-slate-500 group-hover:text-rose-500 transition-colors" />
                        </div>
                        <div>
                          <p className="font-bold text-white group-hover:text-rose-500 transition-colors">{gateway.gateway_name}</p>
                          <p className="text-[10px] font-mono text-slate-500 mt-0.5">UID: {gateway.id.slice(0, 12)}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-2">
                        <div className="w-1.5 h-1.5 bg-indigo-500 rounded-full" />
                        <span className="text-sm font-bold text-slate-300">{gateway.tenant_name}</span>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <StatusBadge status={gateway.status} />
                    </td>
                    <td className="px-8 py-6">
                      <div className="space-y-1">
                        <span className="text-xs font-black uppercase text-white tracking-tighter bg-slate-950 px-2 py-0.5 rounded border border-slate-800">
                            {gateway.subscription_type}
                        </span>
                        <p className="text-sm font-bold text-slate-500 mt-1">
                          ${(gateway.subscription_fee + gateway.server_rental_fee).toLocaleString()}/mo
                        </p>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <div className="flex items-center gap-6">
                        <div className="text-center">
                            <div className="text-xs font-black text-white">{gateway.uptime_percentage || 0}%</div>
                            <div className="text-[10px] font-bold text-slate-600 uppercase tracking-tighter">Uptime</div>
                        </div>
                        <div className="text-center">
                            <div className="text-xs font-black text-white">{(gateway.messages_sent || 0).toLocaleString()}</div>
                            <div className="text-[10px] font-bold text-slate-600 uppercase tracking-tighter">Throughput</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-8 py-6">
                      <button
                        onClick={() => setSelectedGateway(gateway.id)}
                        className="p-2 bg-slate-950 text-slate-500 hover:text-white hover:bg-rose-600 rounded-xl transition-all border border-slate-800"
                      >
                        <ChevronRight className="w-5 h-5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create Gateway Modal */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center z-50 p-6 animate-in fade-in duration-300">
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] shadow-2xl max-w-2xl w-full p-12 relative">
            <button onClick={() => setShowCreateForm(false)} className="absolute top-8 right-8 p-3 bg-slate-950 text-slate-500 hover:text-white rounded-2xl border border-slate-800 transition-all">
                <X className="w-6 h-6" />
            </button>

            <h2 className="text-3xl font-black text-white mb-2">Deploy Gateway</h2>
            <p className="text-slate-500 font-medium mb-10">Configure enterprise dedicated messaging node.</p>
            
            <form onSubmit={handleCreateGateway} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <label className="block text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2 ml-1">Target Tenant ID</label>
                    <input
                    type="text"
                    required
                    value={createForm.tenant_id}
                    onChange={(e) => setCreateForm({...createForm, tenant_id: e.target.value})}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all"
                    />
                </div>
                <div>
                    <label className="block text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2 ml-1">Gateway Alias</label>
                    <input
                    type="text"
                    required
                    value={createForm.gateway_name}
                    onChange={(e) => setCreateForm({...createForm, gateway_name: e.target.value})}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all"
                    />
                </div>
              </div>

              <div>
                <label className="block text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2 ml-1">Billing Cycle</label>
                <select
                  value={createForm.subscription_type}
                  onChange={(e) => setCreateForm({...createForm, subscription_type: e.target.value})}
                  className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 appearance-none cursor-pointer"
                >
                  <option value="monthly">Monthly Subscription</option>
                  <option value="yearly">Yearly Commitment</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-6">
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2 ml-1">Sub Fee ($)</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    value={createForm.subscription_fee}
                    onChange={(e) => setCreateForm({...createForm, subscription_fee: parseFloat(e.target.value)})}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-mono"
                  />
                </div>
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2 ml-1">Rental Fee ($)</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    value={createForm.server_rental_fee}
                    onChange={(e) => setCreateForm({...createForm, server_rental_fee: parseFloat(e.target.value)})}
                    className="w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-mono"
                  />
                </div>
              </div>

              <div className="pt-6">
                <button
                  type="submit"
                  className="w-full py-5 bg-rose-600 hover:bg-rose-500 text-white font-black uppercase tracking-widest rounded-2xl shadow-xl shadow-rose-900/20 transition-all"
                >
                  Initiate Deployment
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ icon: Icon, label, value, color }) {
  const colors = {
    rose:    'from-rose-500/20 to-rose-500/5 text-rose-500 border-rose-500/20',
    emerald: 'from-emerald-500/20 to-emerald-500/5 text-emerald-400 border-emerald-500/20',
    amber:   'from-amber-500/20 to-amber-500/5 text-amber-400 border-amber-500/20',
    slate:   'from-slate-500/20 to-slate-500/5 text-slate-400 border-slate-800',
  };

  return (
    <div className={`bg-gradient-to-br ${colors[color]} border rounded-3xl p-8 transition-all hover:scale-[1.02] cursor-default group`}>
      <div className="flex items-center justify-between mb-4">
        <div className="p-3 bg-slate-950 rounded-2xl border border-white/5">
            <Icon className="w-6 h-6" />
        </div>
        <span className="text-3xl font-black text-white">{value}</span>
      </div>
      <p className="text-[10px] font-black uppercase tracking-widest text-slate-500">{label}</p>
    </div>
  );
}

function StatusBadge({ status }) {
  const getStatusConfig = (status) => {
    switch (status) {
      case 'active':     return { styles: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20', dot: 'bg-emerald-500 shadow-emerald-500/50' };
      case 'suspended':  return { styles: 'bg-amber-500/10 text-amber-400 border-amber-500/20',   dot: 'bg-amber-500 shadow-amber-500/50' };
      case 'terminated': return { styles: 'bg-rose-500/10 text-rose-400 border-rose-500/20',    dot: 'bg-rose-500 shadow-rose-500/50' };
      default:           return { styles: 'bg-slate-500/10 text-slate-400 border-slate-800',       dot: 'bg-slate-500 shadow-slate-500/50' };
    }
  };

  const config = getStatusConfig(status);

  return (
    <span className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-[10px] font-black uppercase border ${config.styles}`}>
      <div className={`w-1.5 h-1.5 rounded-full shadow-[0_0_8px] ${config.dot}`} />
      {status}
    </span>
  );
}

