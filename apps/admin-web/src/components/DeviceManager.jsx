import { useState } from 'react';
import { Smartphone, Battery, Wifi, WifiOff, RefreshCw, Trash2, Search } from 'lucide-react';

export default function DeviceManager() {
    const [devices, setDevices] = useState([
        { id: 1, name: 'Redmi Note 10', phone: '+966551234567', tenant: 'مؤسسة النور للتقنية', status: 'Online', battery: 85, last_seen: 'الآن' },
        { id: 2, name: 'Samsung S21', phone: '+966509988776', tenant: 'بوتيك سارة', status: 'Offline', battery: 12, last_seen: 'منذ ساعتين' },
        { id: 3, name: 'Pixel 6', phone: '+966561122334', tenant: 'عيادات د. خالد', status: 'Online', battery: 98, last_seen: 'منذ دقيقة' },
        { id: 4, name: 'Huawei P30', phone: '+966547788990', tenant: 'غير معروف', status: 'Online', battery: 45, last_seen: 'منذ 5 دقائق' },
    ]);

    return (
        <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">

            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
                <div>
                    <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                        <Smartphone className="w-6 h-6 text-rose-500" />
                        Connected Devices
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">Monitor physical devices linked to the platform.</p>
                </div>
                <div className="flex gap-3">
                    <div className="relative">
                        <Search className="w-4 h-4 text-slate-500 absolute left-3 top-3" />
                        <input
                            type="text"
                            placeholder="Search phone..."
                            className="bg-slate-900 border border-slate-800 rounded-xl pl-10 pr-4 py-2.5 text-sm text-white focus:ring-1 focus:ring-rose-500 outline-none w-64"
                        />
                    </div>
                    <button className="p-2.5 bg-slate-800 hover:bg-slate-700 rounded-xl text-slate-300 transition-colors">
                        <RefreshCw className="w-4 h-4" />
                    </button>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {devices.map((device) => (
                    <div key={device.id} className="glass-card p-5 rounded-3xl border border-slate-800 bg-slate-900/40 hover:border-slate-700 transition-all group">

                        <div className="flex justify-between items-start mb-4">
                            <div className="flex items-center gap-3">
                                <div className={`w-12 h-12 rounded-2xl flex items-center justify-center text-xl ${device.status === 'Online' ? 'bg-emerald-500/10 text-emerald-500' : 'bg-slate-800 text-slate-500'
                                    }`}>
                                    <Smartphone className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="font-bold text-white text-base">{device.name}</h3>
                                    <p className="text-xs text-slate-500 font-mono">{device.phone}</p>
                                </div>
                            </div>
                            <div className={`px-2 py-1 rounded-lg text-[10px] font-bold uppercase flex items-center gap-1 ${device.status === 'Online' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'
                                }`}>
                                {device.status === 'Online' ? <Wifi className="w-3 h-3" /> : <WifiOff className="w-3 h-3" />}
                                {device.status}
                            </div>
                        </div>

                        <div className="space-y-3 bg-slate-950/50 rounded-xl p-4 border border-slate-800/50 mb-4">
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">Tenant Owner</span>
                                <span className="text-slate-300 font-medium">{device.tenant}</span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">Battery Level</span>
                                <span className={`font-bold flex items-center gap-1 ${device.battery < 20 ? 'text-rose-400' : 'text-emerald-400'
                                    }`}>
                                    <Battery className="w-3 h-3" /> {device.battery}%
                                </span>
                            </div>
                            <div className="flex justify-between text-xs">
                                <span className="text-slate-500">Last Heartbeat</span>
                                <span className="text-slate-400">{device.last_seen}</span>
                            </div>
                        </div>

                        <button className="w-full py-2.5 rounded-xl border border-rose-500/20 text-rose-500 hover:bg-rose-500 hover:text-white transition-all text-xs font-bold flex items-center justify-center gap-2">
                            <Trash2 className="w-4 h-4" /> Unlink Device
                        </button>

                    </div>
                ))}
            </div>

        </div>
    );
}
