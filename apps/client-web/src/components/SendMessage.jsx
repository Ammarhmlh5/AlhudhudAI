import React, { useState, useEffect } from 'react';
import { Send, MessageSquare, AlertTriangle, DollarSign, Hash, Loader } from 'lucide-react';
import apiClient from '../api/axios';

const SendMessage = () => {
    const [phoneNumber, setPhoneNumber] = useState('');
    const [messageText, setMessageText] = useState('');
    const [sendMethod, setSendMethod] = useState('package'); // 'package' or 'hudud'
    const [costInfo, setCostInfo] = useState(null);
    const [calculating, setCalculating] = useState(false);
    const [sending, setSending] = useState(false);
    const [sendStatus, setSendStatus] = useState(null);

    // Calculate cost whenever message text changes
    useEffect(() => {
        if (messageText.trim().length > 0) {
            calculateCost();
        } else {
            setCostInfo(null);
        }
    }, [messageText, sendMethod]);

    const calculateCost = async () => {
        setCalculating(true);
        try {
            const response = await apiClient.post('/api/v1/messaging/calculate-cost', {
                message_text: messageText,
                send_method: sendMethod
            });
            setCostInfo(response.data);
        } catch (error) {
            console.error('Error calculating cost:', error);
            setCostInfo(null);
        } finally {
            setCalculating(false);
        }
    };

    const handleSendMessage = async () => {
        if (!phoneNumber || !messageText) {
            alert('الرجاء إدخال رقم الهاتف ونص الرسالة');
            return;
        }

        setSending(true);
        setSendStatus(null);

        try {
            const response = await apiClient.post('/api/v1/messaging/send', {
                phone_number: phoneNumber,
                message_text: messageText,
                send_method: sendMethod
            });

            setSendStatus('success');
            setPhoneNumber('');
            setMessageText('');
            setCostInfo(null);
        } catch (error) {
            console.error('Error sending message:', error);
            setSendStatus('error');
            alert(error.response?.data?.error || 'حدث خطأ أثناء إرسال الرسالة');
        } finally {
            setSending(false);
        }
    };

    const hasInsufficientBalance = costInfo && costInfo.insufficient_balance;

    return (
        <div className="animate-in fade-in slide-in-from-right-8 duration-500">
            <div className="mb-8">
                <h2 className="text-3xl font-bold">إرسال رسالة</h2>
                <p className="text-slate-400 text-sm mt-1">أرسل رسالة نصية إلى رقم واحد</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Message Form */}
                <div className="lg:col-span-2 space-y-6">
                    <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl">
                        <h3 className="text-lg font-bold mb-6 flex items-center gap-2">
                            <MessageSquare className="w-5 h-5 text-rose-500" />
                            تفاصيل الرسالة
                        </h3>

                        {/* Send Method Selection */}
                        <div className="mb-6">
                            <label className="block text-sm font-bold text-slate-300 mb-3">طريقة الإرسال</label>
                            <div className="flex gap-4">
                                <button
                                    onClick={() => setSendMethod('package')}
                                    className={`flex-1 p-4 rounded-xl border-2 transition-all ${sendMethod === 'package'
                                        ? 'border-rose-500 bg-rose-500/10 text-white'
                                        : 'border-slate-700 bg-slate-800/30 text-slate-400 hover:border-slate-600'
                                        }`}
                                >
                                    <div className="text-sm font-bold">عبر الباقة</div>
                                    <div className="text-xs text-slate-500 mt-1">استخدام رسائل الباقة</div>
                                </button>
                                <button
                                    onClick={() => setSendMethod('hudud')}
                                    className={`flex-1 p-4 rounded-xl border-2 transition-all ${sendMethod === 'hudud'
                                        ? 'border-blue-500 bg-blue-500/10 text-white'
                                        : 'border-slate-700 bg-slate-800/30 text-slate-400 hover:border-slate-600'
                                        }`}
                                >
                                    <div className="text-sm font-bold">بوابة الهدهد</div>
                                    <div className="text-xs text-slate-500 mt-1">الدفع حسب الاستخدام</div>
                                </button>
                            </div>
                        </div>

                        {/* Phone Number */}
                        <div className="mb-6">
                            <label className="block text-sm font-bold text-slate-300 mb-2">
                                رقم الهاتف <span className="text-rose-500">*</span>
                            </label>
                            <input
                                type="text"
                                value={phoneNumber}
                                onChange={(e) => setPhoneNumber(e.target.value)}
                                placeholder="+967 xxx xxx xxx"
                                className="w-full bg-slate-900 border border-slate-700 rounded-xl py-3 px-4 focus:outline-none focus:border-rose-500 transition-colors text-white"
                            />
                        </div>

                        {/* Message Text */}
                        <div className="mb-6">
                            <label className="block text-sm font-bold text-slate-300 mb-2">
                                نص الرسالة <span className="text-rose-500">*</span>
                            </label>
                            <textarea
                                value={messageText}
                                onChange={(e) => setMessageText(e.target.value)}
                                placeholder="اكتب رسالتك هنا..."
                                rows={6}
                                className="w-full bg-slate-900 border border-slate-700 rounded-xl py-3 px-4 focus:outline-none focus:border-rose-500 transition-colors text-white resize-none"
                            />
                            <div className="flex items-center justify-between mt-2">
                                <span className="text-xs text-slate-500">
                                    {messageText.length} حرف
                                </span>
                                {calculating && (
                                    <span className="text-xs text-blue-400 flex items-center gap-1">
                                        <Loader className="w-3 h-3 animate-spin" />
                                        جاري الحساب...
                                    </span>
                                )}
                            </div>
                        </div>

                        {/* Send Button */}
                        <button
                            onClick={handleSendMessage}
                            disabled={sending || !phoneNumber || !messageText || hasInsufficientBalance}
                            className="w-full bg-rose-600 hover:bg-rose-700 text-white font-bold py-3 rounded-lg flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {sending ? (
                                <>
                                    <Loader className="w-5 h-5 animate-spin" />
                                    جاري الإرسال...
                                </>
                            ) : (
                                <>
                                    <Send className="w-5 h-5" />
                                    إرسال الرسالة
                                </>
                            )}
                        </button>

                        {sendStatus === 'success' && (
                            <div className="mt-4 p-4 bg-emerald-500/10 border border-emerald-500/30 rounded-xl text-emerald-400 text-sm text-center">
                                تم إرسال الرسالة بنجاح!
                            </div>
                        )}
                    </div>
                </div>

                {/* Cost Information */}
                <div className="space-y-6">
                    {costInfo && (
                        <>
                            <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl">
                                <h3 className="text-sm font-bold text-slate-400 mb-4 uppercase tracking-wider">
                                    معلومات التكلفة
                                </h3>

                                <div className="space-y-4">
                                    {/* Number of Segments */}
                                    <div className="flex items-center justify-between p-3 rounded-lg bg-slate-800/30">
                                        <div className="flex items-center gap-2">
                                            <Hash className="w-4 h-4 text-blue-400" />
                                            <span className="text-sm text-slate-400">عدد الأجزاء</span>
                                        </div>
                                        <span className="text-lg font-bold text-white">
                                            {costInfo.segments || 1}
                                        </span>
                                    </div>

                                    {/* Expected Cost */}
                                    <div className="flex items-center justify-between p-3 rounded-lg bg-slate-800/30">
                                        <div className="flex items-center gap-2">
                                            <DollarSign className="w-4 h-4 text-emerald-400" />
                                            <span className="text-sm text-slate-400">التكلفة المتوقعة</span>
                                        </div>
                                        <span className="text-lg font-bold text-emerald-400">
                                            {costInfo.currency === 'YER' ? 'ر.ي' : '$'} {costInfo.estimated_cost?.toFixed(2) || '0.00'}
                                        </span>
                                    </div>

                                    {/* Messages to Deduct (for package method) */}
                                    {sendMethod === 'package' && costInfo.messages_to_deduct && (
                                        <div className="flex items-center justify-between p-3 rounded-lg bg-slate-800/30">
                                            <span className="text-sm text-slate-400">رسائل سيتم خصمها</span>
                                            <span className="text-lg font-bold text-white">
                                                {costInfo.messages_to_deduct}
                                            </span>
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* Insufficient Balance Warning */}
                            {hasInsufficientBalance && (
                                <div className="glass-card p-6 border-rose-500/30">
                                    <div className="flex items-start gap-3">
                                        <AlertTriangle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
                                        <div>
                                            <h4 className="text-sm font-bold text-rose-400 mb-2">رصيد غير كافي</h4>
                                            <p className="text-xs text-slate-400 leading-relaxed">
                                                {sendMethod === 'package'
                                                    ? 'لا يوجد رسائل كافية في الباقة. يمكنك استخدام بوابة الهدهد أو شحن رصيدك.'
                                                    : 'رصيدك الحالي غير كافي لإرسال هذه الرسالة. الرجاء شحن رصيدك.'
                                                }
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </>
                    )}

                    {/* Info Card */}
                    <div className="glass-card p-6">
                        <h4 className="text-sm font-bold text-slate-400 mb-3 uppercase tracking-wider">
                            ملاحظات
                        </h4>
                        <ul className="space-y-2 text-xs text-slate-500">
                            <li className="flex items-start gap-2">
                                <span className="text-rose-500 mt-0.5">•</span>
                                <span>الرسائل الطويلة يتم تقسيمها إلى أجزاء متعددة</span>
                            </li>
                            <li className="flex items-start gap-2">
                                <span className="text-rose-500 mt-0.5">•</span>
                                <span>التكلفة تحسب تلقائياً بناءً على عدد الأجزاء</span>
                            </li>
                            <li className="flex items-start gap-2">
                                <span className="text-rose-500 mt-0.5">•</span>
                                <span>استخدام الباقة يخصم من رسائلك المتاحة</span>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SendMessage;
