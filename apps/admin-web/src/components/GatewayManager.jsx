import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Network, Server, Globe, Activity, Power, RefreshCw } from 'lucide-react';

export default function GatewayManager() {
    const { t } = useTranslation();
    const [gateways, setGateways] = useState([
        { id: 1, name: 'Sawa SMSC 01', type: 'SMPP v3.4', region: 'السعودية - الرياض', ip: '10.20.30.41', status: 'Connected', ping: '12ms' },
        { id: 2, name: 'Mobily Direct', type: 'HTTP API', region: 'السعودية - جدة', ip: '192.168.1.55', status: 'Connected', ping: '45ms' },
        { id: 3, name: 'Zain Backup', type: 'SMPP v3.4', region: 'عالمي', ip: '8.8.4.4', status: 'Offline', ping: '--' },
    ]);

    return (
        <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">

            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                        <Network className="w-6 h-6 text-rose-500" />
                        {t('gatewayManager.title')}
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">{t('gatewayManager.subtitle')}</p>
                </div>
                <button className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 rounded-xl text-slate-300 transition-colors text-sm font-bold">
                    <RefreshCw className="w-4 h-4" /> {t('gatewayManager.refreshStatus')}
                </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {gateways.map((gw) => (
                    <div key={gw.id} className="glass-card p-6 rounded-3xl border border-slate-800 bg-slate-900/40 hover:border-slate-700 transition-all group relative">

                        <div className="absolute top-6 right-6">
                            <div className={`w-3 h-3 rounded-full ${gw.status === 'Connected' ? 'bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]' : 'bg-rose-500'}`} />
                        </div>

                        <div className="flex items-center gap-4 mb-6">
                            <div className="w-14 h-14 rounded-2xl bg-slate-950 border border-slate-800 flex items-center justify-center">
                                <Server className="w-7 h-7 text-slate-400 group-hover:text-white transition-colors" />
                            </div>
                            <div>
                                <h3 className="font-bold text-white text-lg">{gw.name}</h3>
                                <span className="text-xs text-slate-500 font-mono flex items-center gap-1">
                                    <Globe className="w-3 h-3" /> {gw.region}
                                </span>
                            </div>
                        </div>

                        <div className="space-y-3 mb-6 bg-slate-950/50 p-4 rounded-xl border border-slate-800/50">
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">{t('gatewayManager.protocol')}</span>
                                <span className="text-rose-400 font-bold">{gw.type}</span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">{t('gatewayManager.hostIp')}</span>
                                <span className="text-slate-300 font-mono">{gw.ip}</span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">{t('gatewayManager.latency')}</span>
                                <span className={`${gw.status === 'Connected' ? 'text-emerald-400' : 'text-slate-600'} font-bold`}>{gw.ping}</span>
                            </div>
                        </div>

                        <div className="flex gap-2">
                            <button className="flex-1 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold transition-colors">
                                {t('gatewayManager.configure')}
                            </button>
                            <button className="p-2 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-500 transition-colors">
                                <Power className="w-4 h-4" />
                            </button>
                        </div>

                    </div>
                ))}

                {/* Add New Logic Placeholder */}
                <button className="border-2 border-dashed border-slate-800 hover:border-rose-500/50 rounded-3xl p-6 flex flex-col items-center justify-center gap-4 text-slate-500 hover:text-rose-400 transition-colors group">
                    <div className="w-16 h-16 rounded-full bg-slate-900 flex items-center justify-center group-hover:scale-110 transition-transform">
                        <Plus className="w-8 h-8" />
                    </div>
                    <span className="font-bold text-sm">{t('gatewayManager.addNewGateway')}</span>
                </button>
            </div>

        </div>
    );
}

function Plus({ className }) {
    return (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
    );
}
