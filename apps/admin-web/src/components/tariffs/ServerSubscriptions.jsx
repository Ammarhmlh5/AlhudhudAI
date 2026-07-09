import { Server, HardDrive, Cpu, MemoryStick, Clock, Shield, Activity } from 'lucide-react';

const stats = [
    { label: 'السيرفرات المُدارة', value: '—', icon: Server, color: 'text-violet-400', bg: 'bg-violet-500/10 border-violet-500/20' },
    { label: 'سيرفرات نشطة', value: '—', icon: Activity, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20' },
    { label: 'إجمالي التخزين', value: '—', icon: HardDrive, color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
    { label: 'متوسط استخدام CPU', value: '—', icon: Cpu, color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
];

const serverTiers = [
    {
        name: 'سيرفر مُشترك - صغير',
        specs: '2 vCPU · 4GB RAM · 50GB SSD',
        capacity: '10,000 عملية / يوم',
        tenants: 'حتى 5 مستأجرين',
        badge: 'مشترك',
        badgeColor: 'bg-slate-700 text-slate-300',
        color: 'text-slate-400',
    },
    {
        name: 'سيرفر مُشترك - متوسط',
        specs: '4 vCPU · 8GB RAM · 100GB SSD',
        capacity: '50,000 عملية / يوم',
        tenants: 'حتى 20 مستأجراً',
        badge: 'مشترك',
        badgeColor: 'bg-sky-600/30 text-sky-300',
        color: 'text-sky-400',
    },
    {
        name: 'سيرفر خاص - متقدم',
        specs: '8 vCPU · 16GB RAM · 500GB SSD',
        capacity: '200,000 عملية / يوم',
        tenants: 'مستأجر واحد حصري',
        badge: 'خاص',
        badgeColor: 'bg-violet-600/30 text-violet-300',
        color: 'text-violet-400',
    },
    {
        name: 'سيرفر مؤسسي - بريميوم',
        specs: '16+ vCPU · 32GB+ RAM · NVMe',
        capacity: 'غير محدودة',
        tenants: 'عزل مؤسسي كامل',
        badge: 'بريميوم',
        badgeColor: 'bg-amber-600/30 text-amber-300',
        color: 'text-amber-400',
    },
];

export default function ServerSubscriptions() {
    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-3 duration-500" dir="rtl">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-3 rounded-2xl bg-violet-500/10 border border-violet-500/20">
                        <Server className="w-6 h-6 text-violet-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white">اشتراك السيرفرات</h3>
                        <p className="text-slate-400 text-sm mt-0.5">إدارة اشتراكات السيرفرات المخصصة وخطط الموارد</p>
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

            {/* Server Tiers Table */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
                    <h4 className="text-white font-bold">فئات اشتراكات السيرفرات</h4>
                    <span className="text-xs text-slate-500 bg-slate-800 px-3 py-1 rounded-full">معاينة / غير مفعّل</span>
                </div>
                <div className="divide-y divide-slate-800/50">
                    {serverTiers.map((tier) => (
                        <div key={tier.name} className="flex items-center justify-between px-6 py-5 hover:bg-slate-800/30 transition-colors group">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-slate-800 group-hover:bg-slate-700 transition-colors">
                                    <Server className={`w-4 h-4 ${tier.color}`} />
                                </div>
                                <div>
                                    <div className="flex items-center gap-2">
                                        <p className="text-white font-bold text-sm">{tier.name}</p>
                                        <span className={`text-xs px-2 py-0.5 rounded-full font-bold ${tier.badgeColor}`}>{tier.badge}</span>
                                    </div>
                                    <p className="text-slate-500 text-xs mt-0.5 font-mono">{tier.specs}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-6">
                                <div className="text-right hidden md:block">
                                    <p className="text-slate-400 text-xs">الطاقة الاستيعابية</p>
                                    <p className="text-white text-sm font-bold">{tier.capacity}</p>
                                </div>
                                <div className="text-right">
                                    <p className="text-slate-400 text-xs">العزل</p>
                                    <p className={`text-sm font-bold ${tier.color}`}>{tier.tenants}</p>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Resource Breakdown */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {[
                    { icon: Cpu, label: 'المعالج (vCPU)', desc: 'تخصيص وحدات المعالجة الافتراضية لكل اشتراك', color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
                    { icon: MemoryStick, label: 'الذاكرة (RAM)', desc: 'تحديد سعة الذاكرة العشوائية المخصصة', color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
                    { icon: HardDrive, label: 'التخزين (SSD)', desc: 'إدارة مساحة التخزين المخصصة والنسخ الاحتياطي', color: 'text-violet-400', bg: 'bg-violet-500/10 border-violet-500/20' },
                ].map((item) => (
                    <div key={item.label} className={`p-5 rounded-2xl border ${item.bg} flex items-start gap-3`}>
                        <div className="p-2 rounded-xl bg-slate-900/50 shrink-0">
                            <item.icon className={`w-4 h-4 ${item.color}`} />
                        </div>
                        <div>
                            <p className={`font-bold text-sm ${item.color}`}>{item.label}</p>
                            <p className="text-slate-500 text-xs mt-1 leading-relaxed">{item.desc}</p>
                        </div>
                    </div>
                ))}
            </div>

            {/* Info Notice */}
            <div className="flex items-start gap-4 p-5 rounded-2xl bg-violet-500/5 border border-violet-500/15 border-dashed">
                <div className="p-3 rounded-xl bg-violet-500/10 shrink-0 mt-0.5">
                    <Shield className="w-5 h-5 text-violet-400" />
                </div>
                <div>
                    <p className="text-violet-300 font-bold text-sm">هذا القسم قيد الإعداد</p>
                    <p className="text-slate-500 text-xs mt-1">
                        سيتمكن المشرف من إنشاء خطط اشتراك للسيرفرات المخصصة، تحديد الموارد (CPU, RAM, Storage) لكل خطة، ربطها بالشركات المشتركة، وإدارة دورات الفوترة الشهرية والسنوية.
                    </p>
                </div>
            </div>
        </div>
    );
}
