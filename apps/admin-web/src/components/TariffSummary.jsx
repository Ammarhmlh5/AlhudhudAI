import { useState, useEffect } from 'react';
import { FileText, TrendingUp, Globe, Smartphone } from 'lucide-react';
import { API_BASE_URL } from '../config';

export default function TariffSummary() {
    const [tariffs, setTariffs] = useState([]);
    const [loading, setLoading] = useState(true);
    const [stats, setStats] = useState({
        total_tariffs: 0,
        active_tariffs: 0,
        avg_rate: 0
    });

    useEffect(() => {
        fetchTariffs();
    }, []);

    const fetchTariffs = async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/tariffs`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
            });
            
            if (res.ok) {
                const data = await res.json();
                setTariffs(data.slice(0, 3)); // Show only top 3
                
                // Calculate stats
                const active = data.filter(t => t.is_active).length;
                const avgRate = data.length > 0 
                    ? data.reduce((sum, t) => sum + parseFloat(t.cost_per_message || 0), 0) / data.length 
                    : 0;
                
                setStats({
                    total_tariffs: data.length,
                    active_tariffs: active,
                    avg_rate: avgRate
                });
            }
        } catch (err) {
            console.error("Failed to fetch tariffs", err);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
                <div className="text-center py-8 text-slate-500">Loading tariffs...</div>
            </div>
        );
    }

    return (
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-2xl bg-rose-500/10 flex items-center justify-center">
                        <FileText className="w-6 h-6 text-rose-500" />
                    </div>
                    <div>
                        <h2 className="text-xl font-black text-white">Active Tariffs</h2>
                        <p className="text-xs text-slate-500">Current pricing rates</p>
                    </div>
                </div>
                <div className="flex gap-4">
                    <div className="text-right">
                        <div className="text-2xl font-black text-white">{stats.active_tariffs}</div>
                        <div className="text-xs text-slate-500">Active</div>
                    </div>
                    <div className="text-right">
                        <div className="text-2xl font-black text-emerald-400">${stats.avg_rate.toFixed(3)}</div>
                        <div className="text-xs text-slate-500">Avg Rate</div>
                    </div>
                </div>
            </div>

            {tariffs.length === 0 ? (
                <div className="text-center py-8 text-slate-500">
                    <FileText className="w-12 h-12 mx-auto mb-3 opacity-30" />
                    <p>No tariffs configured yet</p>
                </div>
            ) : (
                <div className="space-y-3">
                    {tariffs.map((tariff) => (
                        <div
                            key={tariff.id}
                            className="bg-slate-950 border border-slate-800 rounded-2xl p-4 flex items-center justify-between hover:border-slate-700 transition-colors"
                        >
                            <div className="flex items-center gap-4">
                                <div className="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center">
                                    {tariff.country_code === 'YE' ? (
                                        <Smartphone className="w-5 h-5 text-emerald-500" />
                                    ) : (
                                        <Globe className="w-5 h-5 text-blue-500" />
                                    )}
                                </div>
                                <div>
                                    <div className="font-bold text-white">
                                        {tariff.country_code} - {tariff.network_operator || 'All Networks'}
                                    </div>
                                    <div className="text-xs text-slate-500">
                                        {tariff.message_type || 'Standard'} Message
                                    </div>
                                </div>
                            </div>
                            <div className="text-right">
                                <div className="font-mono text-emerald-400 font-bold">
                                    ${parseFloat(tariff.cost_per_message || 0).toFixed(4)}
                                </div>
                                <div className={`text-xs ${tariff.is_active ? 'text-emerald-500' : 'text-slate-500'}`}>
                                    {tariff.is_active ? 'Active' : 'Inactive'}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            <div className="mt-6 pt-6 border-t border-slate-800">
                <a
                    href="/tariffs"
                    className="text-sm text-rose-400 hover:text-rose-300 font-bold flex items-center gap-2 transition-colors"
                >
                    View All Tariffs
                    <TrendingUp className="w-4 h-4" />
                </a>
            </div>
        </div>
    );
}
