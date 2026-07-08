import { useState } from 'react';
import { FileText, Smartphone, Globe, Router, Server } from 'lucide-react';
import PhonePackages from './tariffs/PhonePackages';
import MyGatewayPlans from './tariffs/MyGatewayPlans';
import HudhudPhones from './tariffs/HudhudPhones';
import HudhudGateways from './tariffs/HudhudGateways';
import ServerSubscriptions from './tariffs/ServerSubscriptions';

const TABS = [
    {
        id: 'phone-packages',
        label: 'باقات الهواتف',
        icon: Smartphone,
        component: PhonePackages,
    },
    {
        id: 'gateway-plans',
        label: 'خطط بوابتي',
        icon: Globe,
        component: MyGatewayPlans,
    },
    {
        id: 'hudhud-phones',
        label: 'هواتف الهدهد',
        icon: Smartphone,
        component: HudhudPhones,
    },
    {
        id: 'hudhud-gateways',
        label: 'بوابات الهدهد',
        icon: Router,
        component: HudhudGateways,
    },
    {
        id: 'server-subscriptions',
        label: 'اشتراك السيرفرات',
        icon: Server,
        component: ServerSubscriptions,
    },
];

export default function TariffManager() {
    const [activeTab, setActiveTab] = useState('phone-packages');

    const ActiveComponent = TABS.find((t) => t.id === activeTab)?.component;

    return (
        <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500" dir="rtl">
            {/* Page Header */}
            <div>
                <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                    <FileText className="w-6 h-6 text-rose-500" />
                    إدارة التعرفات والخطط
                </h2>
                <p className="text-slate-400 text-sm mt-1">
                    ضبط أسعار الاشتراكات وخطط الباقات لجميع خدمات منصة الهدهد
                </p>
            </div>

            {/* Sub-Navigation Tabs */}
            <div className="flex gap-1 border-b border-slate-800 pb-0 overflow-x-auto scrollbar-hide">
                {TABS.map((tab) => {
                    const Icon = tab.icon;
                    const isActive = activeTab === tab.id;
                    return (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`
                                relative flex items-center gap-2 px-5 py-3 text-sm font-bold whitespace-nowrap
                                transition-all duration-200 shrink-0
                                ${isActive
                                    ? 'text-rose-400'
                                    : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800/40'
                                }
                                rounded-t-lg
                            `}
                        >
                            <Icon className="w-4 h-4" />
                            {tab.label}
                            {/* Active indicator line */}
                            <span
                                className={`
                                    absolute bottom-0 left-0 w-full h-0.5 rounded-t-full
                                    transition-all duration-300
                                    ${isActive ? 'bg-rose-500 opacity-100' : 'bg-transparent opacity-0'}
                                `}
                            />
                        </button>
                    );
                })}
            </div>

            {/* Tab Content */}
            <div className="min-h-[500px] pt-2">
                {ActiveComponent && <ActiveComponent />}
            </div>
        </div>
    );
}
