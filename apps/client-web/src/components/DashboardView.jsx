import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useSocket } from '../contexts/SocketContext';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

// ── Dual-Arc Speedometer ──────────────────────────────────────────────────────
// Outer arc = sرعة الإرسال (red)
// Inner arc = سرعة المعالجة (blue)
const DualSpeedometer = ({ sendValue = 0, processValue = 0, max = 100 }) => {
    const cx = 110;
    const cy = 115;
    const outerR = 85;
    const innerR = 62;
    const start = 225;
    const sweep = 270;

    const polar = (r, angle) => {
        const rad = ((angle - 90) * Math.PI) / 180;
        return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) };
    };

    const arc = (r, fromAngle, toAngle) => {
        const s = polar(r, fromAngle);
        const e = polar(r, toAngle);
        const large = toAngle - fromAngle > 180 ? 1 : 0;
        return `M ${s.x} ${s.y} A ${r} ${r} 0 ${large} 1 ${e.x} ${e.y}`;
    };

    const sendPct = Math.min(Math.max(sendValue / max, 0), 1);
    const procPct = Math.min(Math.max(processValue / max, 0), 1);
    const sendEnd = start + sendPct * sweep;
    const procEnd = start + procPct * sweep;

    // Ticks for outer ring
    const ticks = Array.from({ length: 11 }, (_, i) => {
        const angle = start + (i / 10) * sweep;
        const ro = polar(outerR + 2, angle);
        const ri = polar(outerR - 11, angle);
        return { ...ro, x2: ri.x, y2: ri.y, major: i % 5 === 0 };
    });

    return (
        <div className="flex flex-col items-center">
            <svg width="220" height="180" viewBox="0 0 220 185">
                {/* Outer track */}
                <path d={arc(outerR, start, start + sweep)} fill="none" stroke="#1e293b" strokeWidth="13" strokeLinecap="round" />
                {/* Inner track */}
                <path d={arc(innerR, start, start + sweep)} fill="none" stroke="#1e293b" strokeWidth="13" strokeLinecap="round" />

                {/* Outer fill — send speed (red) */}
                {sendPct > 0 && (
                    <path d={arc(outerR, start, sendEnd)} fill="none" stroke="#f43f5e" strokeWidth="13" strokeLinecap="round"
                        style={{ filter: 'drop-shadow(0 0 5px #f43f5e88)' }} />
                )}
                {/* Inner fill — process speed (blue) */}
                {procPct > 0 && (
                    <path d={arc(innerR, start, procEnd)} fill="none" stroke="#3b82f6" strokeWidth="13" strokeLinecap="round"
                        style={{ filter: 'drop-shadow(0 0 5px #3b82f688)' }} />
                )}

                {/* Tick marks */}
                {ticks.map((t, i) => (
                    <line key={i} x1={t.x} y1={t.y} x2={t.x2} y2={t.y2}
                        stroke={t.major ? '#475569' : '#1e293b'} strokeWidth={t.major ? 2 : 1} />
                ))}

                {/* Center values */}
                <text x={cx} y={cy - 10} textAnchor="middle" fill="#f43f5e" fontSize="18" fontWeight="bold" fontFamily="monospace">{sendValue}</text>
                <text x={cx} y={cy + 10} textAnchor="middle" fill="#64748b" fontSize="8">سرعة الإرسال</text>
                <text x={cx} y={cy + 28} textAnchor="middle" fill="#3b82f6" fontSize="18" fontWeight="bold" fontFamily="monospace">{processValue}</text>
                <text x={cx} y={cy + 46} textAnchor="middle" fill="#64748b" fontSize="8">سرعة المعالجة</text>
            </svg>

            {/* Legend */}
            <div className="flex items-center gap-4 mt-1 text-xs">
                <span className="flex items-center gap-1">
                    <span className="w-3 h-3 rounded-full bg-rose-500 inline-block" />
                    <span className="text-slate-400">إرسال</span>
                </span>
                <span className="flex items-center gap-1">
                    <span className="w-3 h-3 rounded-full bg-blue-500 inline-block" />
                    <span className="text-slate-400">معالجة</span>
                </span>
            </div>
        </div>
    );
};

// ── StatCard ──────────────────────────────────────────────────────────────────
const StatCard = ({ label, value, color, icon }) => (
    <div className="bg-slate-900 border border-slate-800 p-5 rounded-xl flex items-center justify-between">
        <div>
            <p className="text-slate-400 text-xs mb-1">{label}</p>
            <h3 className={`text-2xl font-bold ${color}`}>{value}</h3>
        </div>
        <div className={`text-2xl ${color} opacity-30`}>{icon}</div>
    </div>
);

// ── DashboardView ─────────────────────────────────────────────────────────────
const DashboardView = () => {
    const [metrics, setMetrics] = useState(null);
    const [usage, setUsage] = useState(null);
    const [loading, setLoading] = useState(true);
    const [sendingMode, setSendingMode] = useState('SIGNAL');
    const { isConnected, incomingMessages } = useSocket();
    const { tenant } = useAuth();

    useEffect(() => {
        const fetchData = async () => {
            try {
                // Fetch metrics
                const mRes = await apiClient.get('/api/v1/tenant/dashboard');
                setMetrics(mRes.data);

                // Fetch usage
                const uRes = await apiClient.get('/api/v1/finance/balance');
                setUsage(uRes.data);

                // Fetch support chat
                const cRes = await apiClient.get('/api/v1/support/chat');
                const ticket = cRes.data;
                if (ticket.replies) {
                    const mapped = ticket.replies.map(m => ({
                        id: m.id,
                        from: m.sender_type === 'TENANT' ? 'user' : 'support',
                        text: m.body,
                        time: new Date(m.created_at).toLocaleTimeString('ar', { hour: '2-digit', minute: '2-digit' })
                    }));
                    setChatMessages(mapped);
                    setActiveTicketId(ticket.id);
                }
            } catch (e) {
                console.error('Fetch failed', e);
            } finally {
                setLoading(false);
            }
        };

        fetchData();
        const iv = setInterval(fetchData, 10000); // 10s for dashboard
        return () => clearInterval(iv);
    }, []);

    // ── Support Chat State ────────────────────────────────────────────
    const [chatMessages, setChatMessages] = useState([]);
    const [activeTicketId, setActiveTicketId] = useState(null);
    const [chatInput, setChatInput] = useState('');
    const chatEndRef = React.useRef(null);

    const sendChatMessage = async (e) => {
        e.preventDefault();
        const text = chatInput.trim();
        if (!text || !activeTicketId) return;

        try {
            const response = await apiClient.post(`/api/v1/support/tickets/${activeTicketId}/reply`, { body: text });

            setChatInput('');
            // Refresh messages immediately
            const cRes = await apiClient.get('/api/v1/support/chat');
            const ticket = cRes.data;
            if (ticket.replies) {
                const mapped = ticket.replies.map(m => ({
                    id: m.id,
                    from: m.sender_type === 'TENANT' ? 'user' : 'support',
                    text: m.body,
                    time: new Date(m.created_at).toLocaleTimeString('ar', { hour: '2-digit', minute: '2-digit' })
                }));
                setChatMessages(mapped);
            }
        } catch (error) {
            console.error('Failed to send message:', error);
        }
    };

    // Auto-scroll chat to bottom
    React.useEffect(() => { chatEndRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [chatMessages]);

    const messages = metrics?.messages || {};
    const devices = metrics?.devices || { connected: 0, total: 0 };

    // Protected against NaN which causes fatal SVG DOM render crashes
    const safeSendRate = Number(messages.send_rate ?? Math.round((messages.total_sent || 0) / 60)) || 0;
    const safeProcRate = Number(messages.process_rate ?? Math.round((messages.total_delivered || 0) / 60)) || 0;
    const sendSpeed = Math.min(safeSendRate, 100);
    const procSpeed = Math.min(safeProcRate, 100);

    // Usage metrics based on Sovereign dual-wallet system
    const remainingMsgs = usage?.quota_balance || 0;
    const balance = usage?.cash_balance || 0;

    return (
        <div className="space-y-6">

            {/* ── Header ── */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-3xl font-bold text-white tracking-tight">
                        {tenant?.business_name || 'الرئيسية'}
                    </h2>
                    <div className="flex items-center gap-2 mt-1">
                        <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}`} />
                        <span className="text-slate-500 text-xs font-bold uppercase tracking-widest">{isConnected ? 'Server Connected' : 'Server Disconnected'}</span>
                    </div>
                </div>

                <div className="flex items-center gap-4">
                    {/* Sending Mode Toggle */}
                    <div className="flex flex-col items-center">
                        <button 
                            onClick={() => setSendingMode(sendingMode === 'SIGNAL' ? 'PUSH' : 'SIGNAL')}
                            className={`px-8 py-3 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-lg ${
                                sendingMode === 'SIGNAL' 
                                ? 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-emerald-900/20' 
                                : 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-900/20'
                            }`}
                        >
                            {sendingMode === 'SIGNAL' ? 'SIGNAL' : 'PUSH'}
                        </button>
                        <span className={`text-[10px] font-black uppercase mt-1 tracking-widest ${sendingMode === 'SIGNAL' ? 'text-emerald-500' : 'text-rose-500'}`}>
                            {sendingMode === 'SIGNAL' ? 'وضع السجنل' : 'وضع البوش'}
                        </span>
                    </div>
                </div>
            </div>

            {/* ── Stat Cards (4 consolidated cards) ── */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard label="قائمة الانتظار" value={Number(messages.total_waiting || 0).toLocaleString()} color="text-rose-500" icon="⏳" />
                <StatCard label="الرصيد / الباقة" value={`${Number(balance || 0).toFixed(2)} ر.س / ${Number(remainingMsgs || 0).toLocaleString()}`} color="text-emerald-500" icon="💰" />
                <StatCard label="المرسل (النجاح)" value={`${Number(messages.total_sent || 0).toLocaleString()} (${Number(messages.success_rate || 0).toFixed(1)}%)`} color="text-amber-500" icon="✅" />
                <StatCard label="حالة الهواتف" value={`${Number(devices.connected || 0)} / ${Number(devices.total || 0)}`} color="text-blue-500" icon="📱" />
            </div>

            {/* ── Main Row: Live Feed  |  Right Panel ── */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

                {/* Center: Live Feed (takes 2/3 width) */}
                <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden flex flex-col">
                    <div className="p-5 border-b border-slate-800">
                        <h3 className="text-white font-bold">الرسائل الواردة (Live Feed)</h3>
                        <p className="text-slate-500 text-xs mt-0.5">تحديث فوري للرسائل المستلمة</p>
                    </div>
                    <div className="flex-1 p-4 space-y-2 min-h-[300px] overflow-y-auto">
                        {incomingMessages.length === 0 ? (
                            <div className="flex flex-col items-center justify-center h-full py-20 text-center">
                                <span className="text-4xl mb-3">📭</span>
                                <p className="text-slate-600 italic text-sm">لا توجد رسائل واردة حالياً...</p>
                            </div>
                        ) : (
                            incomingMessages.map((msg) => (
                                <div key={msg.id} className="p-4 bg-slate-950 border border-slate-800 rounded-lg flex justify-between items-center">
                                    <div>
                                        <p className="text-sm font-bold text-white">{msg.phone_number}</p>
                                        <p className="text-xs text-slate-500">{msg.message_content}</p>
                                    </div>
                                    <span className="text-[10px] text-slate-600 font-mono">{new Date().toLocaleTimeString()}</span>
                                </div>
                            ))
                        )}
                    </div>
                </div>

                {/* Right Panel: Gauge + Support (takes 1/3 width) */}
                <div className="space-y-4">

                    {/* Dual Speedometer */}
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col items-center">
                        <h4 className="text-white font-bold text-sm mb-3">مؤشرات الأداء</h4>
                        <DualSpeedometer sendValue={sendSpeed} processValue={procSpeed} max={100} />
                        <p className="text-slate-600 text-[10px] mt-2">وحدة القياس: رسالة / دقيقة</p>
                    </div>

                    {/* ── Support Chat ── */}
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden flex flex-col" style={{ minHeight: '280px' }}>
                        {/* Header */}
                        <div className="px-4 py-3 border-b border-slate-800 flex items-center gap-2 bg-slate-800/40">
                            <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                            <span className="text-white font-bold text-sm">التواصل مع الدعم الفني</span>
                        </div>

                        {/* Messages area */}
                        <div className="flex-1 p-3 space-y-2 overflow-y-auto" style={{ maxHeight: '220px' }}>
                            {chatMessages.map(msg => (
                                <div key={msg.id} className={`flex ${msg.from === 'user' ? 'justify-end' : 'justify-start'}`}>
                                    <div className={`max-w-[80%] px-3 py-2 rounded-2xl text-xs leading-relaxed ${msg.from === 'user'
                                        ? 'bg-rose-600 text-white rounded-tr-sm'
                                        : 'bg-slate-700 text-slate-100 rounded-tl-sm'
                                        }`}>
                                        <p>{msg.text}</p>
                                        <p className={`text-[9px] mt-1 ${msg.from === 'user' ? 'text-rose-200' : 'text-slate-500'}`}>{msg.time}</p>
                                    </div>
                                </div>
                            ))}
                            <div ref={chatEndRef} />
                        </div>

                        {/* Input area */}
                        <form onSubmit={sendChatMessage} className="flex items-center gap-2 p-3 border-t border-slate-800">
                            <input
                                type="text"
                                value={chatInput}
                                onChange={e => setChatInput(e.target.value)}
                                placeholder="اكتب رسالتك..."
                                className="flex-1 bg-slate-800 border border-slate-700 rounded-full py-2 px-4 text-xs focus:outline-none focus:border-rose-500 transition-colors"
                            />
                            <button
                                type="submit"
                                disabled={!chatInput.trim()}
                                className="w-8 h-8 bg-rose-600 hover:bg-rose-500 disabled:opacity-40 rounded-full flex items-center justify-center transition-colors flex-shrink-0"
                            >
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                                    <line x1="22" y1="2" x2="11" y2="13" />
                                    <polygon points="22 2 15 22 11 13 2 9 22 2" />
                                </svg>
                            </button>
                        </form>
                    </div>

                </div>
            </div>

        </div>

    );
};

export default DashboardView;
