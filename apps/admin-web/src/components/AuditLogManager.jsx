import { useState, useEffect } from 'react';
import { ClipboardList, Search, Filter, Shield, User, Globe, Calendar, CheckCircle, XCircle } from 'lucide-react';
import apiClient from '../api/axios';

export default function AuditLogManager() {
    const [entries, setEntries] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [filters, setFilters] = useState({
        action: '',
        resource: '',
        result: '',
    });

    useEffect(() => {
        fetchLogs();
    }, [filters]);

    const fetchLogs = async () => {
        try {
            setLoading(true);
            const queryParams = new URLSearchParams(filters);
            const res = await apiClient.get(`/api/v1/audit?${queryParams.toString()}`);
            setEntries(res.data.entries || []);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="p-8 space-y-8 animate-in fade-in duration-500">
            <div>
                <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                    <ClipboardList className="w-6 h-6 text-rose-500" />
                    System Audit Logs
                </h2>
                <p className="text-slate-400 text-sm mt-1">Track administrative actions, security events, and system changes.</p>
            </div>

            <div className="flex flex-wrap gap-4 items-center bg-slate-900/40 p-6 rounded-3xl border border-slate-800">
                <div className="relative flex-1 min-w-[200px]">
                    <Search className="w-4 h-4 text-slate-500 absolute left-3 top-3" />
                    <input
                        type="text"
                        placeholder="Search by action..."
                        value={filters.action}
                        onChange={(e) => setFilters({ ...filters, action: e.target.value })}
                        className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-10 pr-4 py-2 text-sm text-white focus:ring-1 focus:ring-rose-500 outline-none"
                    />
                </div>
                <select
                    value={filters.resource}
                    onChange={(e) => setFilters({ ...filters, resource: e.target.value })}
                    className="bg-slate-950 border border-slate-800 text-slate-300 text-sm rounded-xl px-4 py-2 outline-none focus:ring-1 focus:ring-rose-500"
                >
                    <option value="">All Resources</option>
                    <option value="TENANT">Tenants</option>
                    <option value="USER">Users</option>
                    <option value="CHANNEL">Channels</option>
                    <option value="CONFIG">Configuration</option>
                    <option value="BALANCE">Balance</option>
                </select>
                <select
                    value={filters.result}
                    onChange={(e) => setFilters({ ...filters, result: e.target.value })}
                    className="bg-slate-950 border border-slate-800 text-slate-300 text-sm rounded-xl px-4 py-2 outline-none focus:ring-1 focus:ring-rose-500"
                >
                    <option value="">All Results</option>
                    <option value="SUCCESS">Success</option>
                    <option value="FAILURE">Failure</option>
                </select>
            </div>

            <div className="glass-card overflow-hidden border border-slate-800 bg-slate-900/40 rounded-3xl">
                <div className="overflow-x-auto w-full">
<table className="w-full text-left">
                    <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                        <tr>
                            <th className="px-6 py-4">Event Info</th>
                            <th className="px-6 py-4">Resource</th>
                            <th className="px-6 py-4">Actor</th>
                            <th className="px-6 py-4">Result</th>
                            <th className="px-6 py-4">Timestamp</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800 text-slate-300">
                        {loading ? (
                            <tr><td colSpan="5" className="px-6 py-8 text-center text-slate-500">Loading logs...</td></tr>
                        ) : entries.length === 0 ? (
                            <tr><td colSpan="5" className="px-6 py-8 text-center text-slate-500">No audit entries found.</td></tr>
                        ) : entries.map((entry) => (
                            <tr key={entry.id} className="hover:bg-slate-800/30 transition-colors group">
                                <td className="px-6 py-4">
                                    <div className="flex items-center gap-3">
                                        <div className="w-8 h-8 rounded-lg bg-slate-800 flex items-center justify-center text-rose-500 font-bold">
                                            {entry.action.charAt(0)}
                                        </div>
                                        <div>
                                            <div className="font-bold text-white text-sm">{entry.action}</div>
                                            <div className="text-[10px] text-slate-500 font-mono tracking-tighter">ID: {entry.resource_id || 'N/A'}</div>
                                        </div>
                                    </div>
                                </td>
                                <td className="px-6 py-4 text-xs font-medium text-slate-400">
                                    <span className="bg-slate-800 px-2 py-1 rounded text-slate-300">{entry.resource}</span>
                                </td>
                                <td className="px-6 py-4">
                                    <div className="flex items-center gap-2 text-xs">
                                        <User className="w-3 h-3 text-slate-500" />
                                        <span>{entry.user_id || 'System'}</span>
                                    </div>
                                    <div className="flex items-center gap-2 text-[10px] text-slate-500 mt-0.5">
                                        <Globe className="w-3 h-3" />
                                        <span>{entry.ip_address}</span>
                                    </div>
                                </td>
                                <td className="px-6 py-4">
                                    <span className={`flex items-center gap-1 text-[10px] font-bold uppercase ${entry.result === 'SUCCESS' ? 'text-emerald-400' : 'text-rose-400'}`}>
                                        {entry.result === 'SUCCESS' ? <CheckCircle className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                                        {entry.result}
                                    </span>
                                </td>
                                <td className="px-6 py-4 text-xs text-slate-400">
                                    <div className="flex items-center gap-1">
                                        <Calendar className="w-3 h-3" />
                                        {new Date(entry.created_at).toLocaleString()}
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
</div>
            </div>
        </div>
    );
}
