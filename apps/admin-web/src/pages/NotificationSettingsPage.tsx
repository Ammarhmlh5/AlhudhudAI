import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import apiClient from '../api/axios';

interface NotificationSetting {
    id: string;
    notification_type: string;
    is_enabled: boolean;
    config: Record<string, any>;
}

interface NotificationTemplate {
    id: string;
    notification_type: string;
    title_template: string;
    body_template: string;
    is_default: boolean;
}

const notificationTypes = [
    { key: 'reconnection_alert', label: 'إشعار إعادة الاتصال', description: 'إشعار عند عودة الهاتف بعد انقطاع' },
    { key: 'package_expiry_warning', label: 'تحذير انتهاء الباقة', description: 'إشعار قبل انتهاء الباقة' },
    { key: 'package_expiry_alert', label: 'إشعار انتهاء الباقة', description: 'إشعار عند انتهاء الباقة' },
    { key: 'low_balance_warning', label: 'تحذير انخفاض الرصيد', description: 'إشعار عند انخفاض الرصيد' },
    { key: 'message_pending_alert', label: 'رسائل معلقة', description: 'إشعار بوجود رسائل معلقة' },
    { key: 'delivery_failed_alert', label: 'فشل التسليم', description: 'إشعار عند فشل تسليم الرسائل' },
    { key: 'system_update', label: 'تحديثات النظام', description: 'إشعارات تحديثات النظام' },
    { key: 'new_message', label: 'رسائل جديدة', description: 'إشعار مع كل رسالة جديدة' },
];

const NotificationSettingsPage: React.FC = () => {
    const { t } = useTranslation();
    const [settings, setSettings] = useState<NotificationSetting[]>([]);
    const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [activeTab, setActiveTab] = useState<'settings' | 'templates' | 'logs'>('settings');
    const [editingTemplate, setEditingTemplate] = useState<NotificationTemplate | null>(null);

    useEffect(() => {
        fetchSettings();
        fetchTemplates();
    }, []);

    const fetchSettings = async () => {
        try {
            const response = await apiClient.get('/api/v1/admin/notifications/settings');
            setSettings(response.data);
        } catch (error) {
            console.error('Failed to fetch settings:', error);
        } finally {
            setLoading(false);
        }
    };

    const fetchTemplates = async () => {
        try {
            const response = await apiClient.get('/api/v1/admin/notifications/templates');
            setTemplates(response.data);
        } catch (error) {
            console.error('Failed to fetch templates:', error);
        }
    };

    const updateSetting = async (notificationType: string, isEnabled: boolean) => {
        setSaving(true);
        try {
            await apiClient.put(
                `/api/v1/admin/notifications/settings/${notificationType}`,
                { is_enabled: isEnabled, config: {} }
            );
            await fetchSettings();
        } catch (error) {
            console.error('Failed to update setting:', error);
        } finally {
            setSaving(false);
        }
    };

    const updateTemplate = async (notificationType: string, titleTemplate: string, bodyTemplate: string) => {
        setSaving(true);
        try {
            await apiClient.put(
                `/api/v1/admin/notifications/templates/${notificationType}`,
                { title_template: titleTemplate, body_template: bodyTemplate }
            );
            await fetchTemplates();
            setEditingTemplate(null);
        } catch (error) {
            console.error('Failed to update template:', error);
        } finally {
            setSaving(false);
        }
    };

    const getSetting = (notificationType: string): NotificationSetting | undefined => {
        return settings.find(s => s.notification_type === notificationType);
    };

    const getTemplate = (notificationType: string): NotificationTemplate | undefined => {
        return templates.find(t => t.notification_type === notificationType);
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen">
                <div className="text-lg">{t('common.loading')}</div>
            </div>
        );
    }

    return (
        <div className="p-6">
            <h1 className="text-2xl font-bold mb-6">{t('notifications.settings.title')}</h1>

            {/* Tabs */}
            <div className="flex gap-4 mb-6 border-b border-gray-700">
                <button
                    onClick={() => setActiveTab('settings')}
                    className={`px-4 py-2 ${activeTab === 'settings' ? 'border-b-2 border-yellow-500 text-yellow-500' : 'text-gray-400'}`}
                >
                    {t('notifications.settings.tabs.settings')}
                </button>
                <button
                    onClick={() => setActiveTab('templates')}
                    className={`px-4 py-2 ${activeTab === 'templates' ? 'border-b-2 border-yellow-500 text-yellow-500' : 'text-gray-400'}`}
                >
                    {t('notifications.settings.tabs.templates')}
                </button>
            </div>

            {/* Settings Tab */}
            {activeTab === 'settings' && (
                <div className="bg-gray-800 rounded-lg p-6">
                    <h2 className="text-lg font-semibold mb-4">{t('notifications.settings.subtitle')}</h2>
                    <div className="space-y-4">
                        {notificationTypes.map(({ key, label, description }) => {
                            const setting = getSetting(key);
                            return (
                                <div key={key} className="flex items-center justify-between p-4 bg-gray-700 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{label}</h3>
                                        <p className="text-sm text-gray-400">{description}</p>
                                    </div>
                                    <label className="relative inline-flex items-center cursor-pointer">
                                        <input
                                            type="checkbox"
                                            checked={setting?.is_enabled ?? false}
                                            onChange={(e) => updateSetting(key, e.target.checked)}
                                            disabled={saving}
                                            className="sr-only peer"
                                        />
                                        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-yellow-500"></div>
                                    </label>
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {/* Templates Tab */}
            {activeTab === 'templates' && (
                <div className="bg-gray-800 rounded-lg p-6">
                    <h2 className="text-lg font-semibold mb-4">{t('notifications.templates.title')}</h2>
                    <div className="grid gap-4">
                        {notificationTypes.map(({ key, label }) => {
                            const template = getTemplate(key);
                            return (
                                <div key={key} className="p-4 bg-gray-700 rounded-lg">
                                    <div className="flex justify-between items-center mb-2">
                                        <h3 className="font-medium">{label}</h3>
                                        <button
                                            onClick={() => setEditingTemplate(template || { id: '', notification_type: key, title_template: '', body_template: '', is_default: false } as any)}
                                            className="px-3 py-1 bg-yellow-500 text-black rounded text-sm hover:bg-yellow-400"
                                        >
                                            {t('common.edit')}
                                        </button>
                                    </div>
                                    {template && (
                                        <div className="text-sm text-gray-400">
                                            <p><span className="text-gray-500">العنوان:</span> {template.title_template}</p>
                                            <p><span className="text-gray-500">المحتوى:</span> {template.body_template}</p>
                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {/* Edit Template Modal */}
            {editingTemplate && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-gray-800 rounded-lg p-6 w-full max-w-lg">
                        <h2 className="text-xl font-bold mb-4">
                            {t('notifications.templates.edit')} - {editingTemplate.notification_type}
                        </h2>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm mb-1">قالب العنوان</label>
                                <input
                                    type="text"
                                    value={editingTemplate.title_template}
                                    onChange={(e) => setEditingTemplate({ ...editingTemplate, title_template: e.target.value })}
                                    className="w-full p-2 bg-gray-700 rounded border border-gray-600"
                                    placeholder="أدخل قالب العنوان"
                                />
                            </div>
                            <div>
                                <label className="block text-sm mb-1">قالب المحتوى</label>
                                <textarea
                                    value={editingTemplate.body_template}
                                    onChange={(e) => setEditingTemplate({ ...editingTemplate, body_template: e.target.value })}
                                    className="w-full p-2 bg-gray-700 rounded border border-gray-600 h-32"
                                    placeholder="أدخل قالب المحتوى"
                                />
                            </div>
                            <div className="flex gap-2 justify-end">
                                <button
                                    onClick={() => setEditingTemplate(null)}
                                    className="px-4 py-2 bg-gray-600 rounded hover:bg-gray-500"
                                >
                                    {t('common.cancel')}
                                </button>
                                <button
                                    onClick={() => updateTemplate(editingTemplate.notification_type, editingTemplate.title_template, editingTemplate.body_template)}
                                    disabled={saving}
                                    className="px-4 py-2 bg-yellow-500 text-black rounded hover:bg-yellow-400"
                                >
                                    {t('common.save')}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
export default NotificationSettingsPage;