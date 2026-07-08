import React, { useState, useEffect } from 'react';
import { 
  Server, Cpu, HardDrive, Network, AlertTriangle, 
  CheckCircle, XCircle, Settings, TrendingUp, Clock, Activity,
  Save, X
} from 'lucide-react';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

export default function GatewayDetails({ gatewayId, onClose }) {
  const [gateway, setGateway] = useState(null);
  const [quotas, setQuotas] = useState(null);
  const [usage, setUsage] = useState(null);
  const [loading, setLoading] = useState(true);
  const [editMode, setEditMode] = useState(false);
  const [quotaForm, setQuotaForm] = useState({});

  useEffect(() => {
    if (gatewayId) {
      fetchGatewayDetails();
    }
  }, [gatewayId]);

  const fetchGatewayDetails = async () => {
    try {
      setLoading(true);
      const gatewayRes = await apiClient.get(`/api/v1/admin/gateways/${gatewayId}`);
      setGateway(gatewayRes.data);

      const quotasRes = await apiClient.get(`/api/v1/admin/gateways/${gatewayId}/quotas`);
      setQuotas(quotasRes.data);
      setQuotaForm(quotasRes.data);

      const usageRes = await apiClient.get(`/api/v1/admin/gateways/${gatewayId}/usage`);
      setUsage(usageRes.data);
    } catch (err) {
      console.error('Failed to fetch gateway details:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateQuotas = async () => {
    try {
      const res = await apiClient.put(`/api/v1/admin/gateways/${gatewayId}/quotas`, quotaForm);
      if (res.status === 200) {
        setQuotas(quotaForm);
        setEditMode(false);
      }
    } catch (err) {
      console.error('Failed to update quotas:', err);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center py-20 space-y-4">
        <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
        <p className="text-slate-500 font-bold animate-pulse">Accessing Secure Node Details...</p>
      </div>
    );
  }

  if (!gateway) {
    return (
      <div className="text-center py-24 bg-slate-950 rounded-3xl border border-slate-800 border-dashed">
        <XCircle className="w-16 h-16 text-slate-800 mx-auto mb-4" />
        <p className="text-slate-500 font-bold">Node not found in local grid.</p>
      </div>
    );
  }

  const getStatusStyles = (status) => {
    switch (status) {
      case 'active':     return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
      case 'suspended':  return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
      case 'terminated': return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
      default:           return 'bg-slate-500/10 text-slate-400 border-slate-800';
    }
  };

  return (
    <div className="space-y-8">
      {/* Gateway Hero Card */}
      <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 shadow-2xl relative overflow-hidden group">
        <div className="absolute top-0 right-0 p-10 opacity-[0.03] group-hover:opacity-[0.08] transition-opacity">
            <Server className="w-48 h-48" />
        </div>

        <div className="flex flex-col md:flex-row md:items-center justify-between gap-8 relative z-10">
          <div className="flex items-center gap-6">
            <div className="p-6 rounded-[2rem] bg-slate-950 border border-slate-800 shadow-inner">
              <Server className="w-10 h-10 text-rose-600" />
            </div>
            <div>
              <div className="flex items-center gap-3 mb-1">
                <h2 className="text-3xl font-black text-white">{gateway.gateway_name}</h2>
                <span className={`px-3 py-1 text-[10px] font-black uppercase rounded-xl border ${getStatusStyles(gateway.status)}`}>
                    {gateway.status}
                </span>
              </div>
              <p className="text-slate-500 font-bold flex items-center gap-2">
                <div className="w-1.5 h-1.5 bg-indigo-500 rounded-full" />
                {gateway.tenant_name}
              </p>
            </div>
          </div>
          
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
            <InfoBox label="Network Type" value={gateway.subscription_type} />
            <InfoBox label="Deployment" value={new Date(gateway.created_at).toLocaleDateString()} />
            <InfoBox label="Latency" value="12ms" />
            <InfoBox label="Uptime" value={`${gateway.uptime_percentage || 0}%`} />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Resource Quotas */}
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 shadow-2xl">
            <div className="flex items-center justify-between mb-10">
                <h3 className="text-xl font-black text-white flex items-center gap-4">
                    <div className="p-2 bg-indigo-500/20 rounded-lg text-indigo-500">
                        <Settings className="w-5 h-5" />
                    </div>
                    Resource Quotas
                </h3>
                {!editMode ? (
                    <button
                        onClick={() => setEditMode(true)}
                        className="px-6 py-2.5 bg-slate-950 border border-slate-800 text-slate-400 hover:text-white rounded-xl text-xs font-black uppercase tracking-widest transition-all"
                    >
                        Modify Node
                    </button>
                ) : (
                    <div className="flex gap-3">
                        <button
                            onClick={handleUpdateQuotas}
                            className="p-2.5 bg-emerald-600 text-white rounded-xl hover:bg-emerald-500 shadow-lg shadow-emerald-900/20 transition-all"
                        >
                            <Save className="w-5 h-5" />
                        </button>
                        <button
                            onClick={() => { setEditMode(false); setQuotaForm(quotas); }}
                            className="p-2.5 bg-slate-800 text-slate-300 rounded-xl hover:bg-slate-700 transition-all"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                )}
            </div>

            {quotas && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <QuotaInput icon={Cpu} label="CPU Capacity" value={editMode ? quotaForm.max_cpu_percent : quotas.max_cpu_percent} unit="%" editMode={editMode} onChange={(val) => setQuotaForm({...quotaForm, max_cpu_percent: val})} />
                    <QuotaInput icon={HardDrive} label="Memory Cap" value={editMode ? quotaForm.max_memory_mb : quotas.max_memory_mb} unit="MB" editMode={editMode} onChange={(val) => setQuotaForm({...quotaForm, max_memory_mb: val})} />
                    <QuotaInput icon={Network} label="Network BW" value={editMode ? quotaForm.max_bandwidth_mbps : quotas.max_bandwidth_mbps} unit="Mbps" editMode={editMode} onChange={(val) => setQuotaForm({...quotaForm, max_bandwidth_mbps: val})} />
                    <QuotaInput icon={TrendingUp} label="Request Limit" value={editMode ? quotaForm.api_rate_limit : quotas.api_rate_limit} unit="R/M" editMode={editMode} onChange={(val) => setQuotaForm({...quotaForm, api_rate_limit: val})} />
                </div>
            )}
          </div>

          {/* Real-time Usage */}
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 shadow-2xl">
            <h3 className="text-xl font-black text-white mb-10 flex items-center gap-4">
                <div className="p-2 bg-emerald-500/20 rounded-lg text-emerald-500">
                    <Activity className="w-5 h-5" />
                </div>
                Live Utilization
            </h3>

            <div className="space-y-8">
                <UsageMeter label="CPU Allocation" used={usage?.cpu_usage || 0} limit={quotas?.max_cpu_percent} unit="%" />
                <UsageMeter label="Memory Buffer" used={usage?.memory_usage_mb || 0} limit={quotas?.max_memory_mb} unit="MB" />
                <UsageMeter label="Bandwidth Flow" used={usage?.bandwidth_usage_mbps || 0} limit={quotas?.max_bandwidth_mbps} unit="Mbps" />
                <UsageMeter label="API Throughput" used={usage?.api_requests_per_min || 0} limit={quotas?.api_rate_limit} unit="R/M" />
            </div>
          </div>
      </div>
    </div>
  );
}

function InfoBox({ label, value }) {
    return (
        <div className="text-center md:text-left">
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-1">{label}</p>
            <p className="text-sm font-bold text-slate-200">{value}</p>
        </div>
    );
}

function QuotaInput({ icon: Icon, label, value, unit, editMode, onChange }) {
  return (
    <div className="bg-slate-950 border border-slate-800 p-5 rounded-3xl group hover:border-indigo-500/30 transition-colors">
      <div className="flex items-center gap-4 mb-3">
        <div className="p-2 bg-slate-900 rounded-xl">
          <Icon className="w-4 h-4 text-slate-500 group-hover:text-indigo-400 transition-colors" />
        </div>
        <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{label}</p>
      </div>
      {editMode ? (
        <div className="flex items-center gap-2">
            <input
            type="number"
            value={value}
            onChange={(e) => onChange(parseFloat(e.target.value))}
            className="w-full bg-slate-900 border border-slate-800 rounded-xl px-4 py-2 text-white focus:outline-none focus:border-indigo-500 font-mono text-sm"
            />
            <span className="text-xs font-bold text-slate-600">{unit}</span>
        </div>
      ) : (
        <div className="text-xl font-black text-white font-mono">
            {value} <span className="text-xs font-bold text-slate-600 ml-1">{unit}</span>
        </div>
      )}
    </div>
  );
}

function UsageMeter({ label, used, limit, unit }) {
  const percentage = limit ? Math.min((used / limit) * 100, 100) : 0;
  const color = percentage >= 90 ? 'bg-rose-500' : percentage >= 75 ? 'bg-amber-500' : 'bg-emerald-500';

  return (
    <div className="space-y-3">
      <div className="flex justify-between items-end">
        <div>
            <p className="text-xs font-black text-white uppercase tracking-widest mb-0.5">{label}</p>
            <p className="text-[10px] font-bold text-slate-500">{used} / {limit} {unit}</p>
        </div>
        <span className={`text-xs font-black font-mono ${percentage >= 90 ? 'text-rose-500' : 'text-slate-400'}`}>
            {percentage.toFixed(1)}%
        </span>
      </div>
      <div className="h-2 bg-slate-950 rounded-full overflow-hidden border border-slate-800 p-0.5">
        <div
          className={`h-full ${color} rounded-full transition-all duration-1000 shadow-[0_0_10px_rgba(0,0,0,0.5)]`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

