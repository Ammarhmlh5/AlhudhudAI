import { Smartphone, Cpu, Battery, Wifi, Clock, Settings2, Shield } from 'lucide-react';

const stats = [
    { label: 'أجهزة الهدهد المسجّلة', value: '—', icon: Smartphone, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20' },
    { label: 'الأجهزة النشطة حالياً', value: '—', icon: Wifi, color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
    { label: 'متوسط البطارية', value: '—', icon: Battery, color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
    { label: 'أجهزة تحتاج مراجعة', value: '—', icon: Settings2, color: 'text-rose-400', bg: 'bg-rose-500/10 border-rose-500/20' },
];

const devicePlans = [
    { name: 'هاتف هدهد - فئة A', specs: 'معالج مزدوج · 3GB RAM', capacity: '500 رسالة / يوم', status: 'إدارة مباشرة', statusColor: 'text-emerald-400 bg-emerald-500/10' },
    { name: 'هاتف هدهد - فئة B', specs: 'معالج رباعي · 6GB RAM', capacity: '2,000 رسالة / يوم', status: 'إدارة مباشرة', statusColor: 'text-sky-400 bg-sky-500/10' },
    { name: 'هاتف هدهد - فئة C', specs: 'معالج ثماني · 8GB RAM', capacity: '5,000 رسالة / يوم', status: 'إدارة مباشرة', statusColor: 'text-violet-400 bg-violet-500/10' },
];

export default function HudhudPhones() {
    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-3 duration-500" dir="rtl">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-3 rounded-2xl bg-emerald-500/10 border border-emerald-500/20">
                        <Smartphone className="w-6 h-6 text-emerald-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white">هواتف الهدهد</h3>
                        <p className="text-slate-400 text-sm mt-0.5">إدارة الهواتف المملوكة لمنصة الهدهد وخطط تعرفاتها</p>
                    </div>
                </div>
                <div className="flex items-center gap-2 px-4 py-2 bg-amber-500/10 border border-amber-500/20 rounded-xl">
                    <Clock className="w-4 h-4 text-amber-400" />
                    <span className="text-amber-400 text-sm font-bold">قيد التطوير</span>
                </div>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {stats.map((stat) => (
                    <div key={stat.label} className={`p-4 rounded-2xl border ${stat.bg} flex flex-col gap-2`}>
                        <stat.icon className={`w-5 h-5 ${stat.color}`} />
                        <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                        <p className="text-slate-400 text-xs leading-relaxed">{stat.label}</p>
                    </div>
                ))}
            </div>

            {/* Device Plans Table */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
                    <h4 className="text-white font-bold">تصنيفات أجهزة الهدهد</h4>
                    <span className="text-xs text-slate-500 bg-slate-800 px-3 py-1 rounded-full">معاينة / غير مفعّل</span>
                </div>
                <div className="divide-y divide-slate-800/50">
                    {devicePlans.map((device) => (
                        <div key={device.name} className="flex items-center justify-between px-6 py-4 hover:bg-slate-800/30 transition-colors group">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-slate-800 group-hover:bg-slate-700 transition-colors">
                                    <Cpu className="w-4 h-4 text-emerald-400" />
                                </div>
                                <div>
                                    <p className="text-white font-bold text-sm">{device.name}</p>
                                    <p className="text-slate-500 text-xs mt-0.5">{device.specs}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-6">
                                <div className="text-right hidden md:block">
                                    <p className="text-slate-400 text-xs">الطاقة الاستيعابية</p>
                                    <p className="text-white text-sm font-bold">{device.capacity}</p>
                                </div>
                                <span className={`text-xs px-3 py-1 rounded-full font-bold ${device.statusColor}`}>{device.status}</span>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Info Notice */}
            <div className="flex items-start gap-4 p-5 rounded-2xl bg-emerald-500/5 border border-emerald-500/15 border-dashed">
                <div className="p-3 rounded-xl bg-emerald-500/10 shrink-0 mt-0.5">
                    <Shield className="w-5 h-5 text-emerald-400" />
                </div>
                <div>
                    <p className="text-emerald-300 font-bold text-sm">هذا القسم قيد الإعداد</p>
                    <p className="text-slate-500 text-xs mt-1">
                        سيتمكن المشرف من إدارة أجهزة الهدهد المملوكة مباشرةً، تخصيصها للشركات، تحديد تعرفات الإيجار والاستخدام، ومتابعة حالة كل جهاز (بطارية، اتصال، حمل رسائل).
                    </p>
                </div>
            </div>
        </div>
    );
}
