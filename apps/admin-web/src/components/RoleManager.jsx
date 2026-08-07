import { useState, useEffect } from 'react';
import { Shield, Plus, Trash2, Edit, CheckSquare, Square, Save, X, Info } from 'lucide-react';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';
import { useTranslation } from 'react-i18next';

export default function RoleManager() {
    const { t } = useTranslation();
    const [roles, setRoles] = useState([]);
    const [permissions, setPermissions] = useState([]);
    const [loading, setLoading] = useState(true);
    const [selectedRole, setSelectedRole] = useState(null);
    const [isEditing, setIsEditing] = useState(false);
    const [formData, setFormData] = useState({ name: '', description: '', permissions: [] });

    useEffect(() => {
        fetchRoles();
        fetchPermissions();
    }, []);

    const fetchRoles = async () => {
        try {
            const res = await apiClient.get('/api/v1/staff/roles');
            if (res.status === 200) setRoles(res.data);
        } catch (err) { console.error(err); }
        finally { setLoading(false); }
    };

    const fetchPermissions = async () => {
        try {
            const res = await apiClient.get('/api/v1/staff/permissions');
            if (res.status === 200) setPermissions(res.data);
        } catch (err) { console.error(err); }
    };

    const handleEdit = (role) => {
        setSelectedRole(role);
        setFormData({
            name: role.name,
            description: role.description,
            permissions: role.permissions ? role.permissions.map(p => p.name) : []
        });
        setIsEditing(true);
    };

    const handleTogglePermission = (name) => {
        setFormData(prev => ({
            ...prev,
            permissions: prev.permissions.includes(name)
                ? prev.permissions.filter(p => p !== name)
                : [...prev.permissions, name]
        }));
    };

    const handleSave = async () => {
        try {
            const url = selectedRole 
                ? '/api/v1/staff/roles/update' 
                : '/api/v1/staff/roles/create';
            const method = selectedRole ? 'put' : 'post';
            const body = selectedRole ? { ...formData, id: selectedRole.id } : formData;

            const res = await apiClient[method](url, body);

            if (res.status === 200) {
                setIsEditing(false);
                fetchRoles();
            }
        } catch (err) { console.error(err); }
    };

    return (
        <div className="p-8 space-y-8 animate-in fade-in duration-500">
            <div className="flex justify-between items-center">
                <div>
                    <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                        <Shield className="w-6 h-6 text-rose-500" />
                        {t('roleManager.title')}
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">{t('roleManager.description')}</p>
                </div>
                <button 
                    onClick={() => { setSelectedRole(null); setFormData({ name: '', description: '', permissions: [] }); setIsEditing(true); }}
                    className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold text-sm transition-all shadow-lg shadow-rose-900/20 flex items-center gap-2"
                >
                    <Plus className="w-4 h-4" /> {t('roleManager.create_role')}
                </button>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* Roles List */}
                <div className="lg:col-span-1 space-y-4">
                    {loading ? (
                        <div className="text-slate-500 text-sm">{t('roleManager.loading_roles')}</div>
                    ) : roles.map(role => (
                        <div 
                            key={role.id}
                            onClick={() => handleEdit(role)}
                            className={`p-6 rounded-3xl border transition-all cursor-pointer group ${
                                selectedRole?.id === role.id ? 'bg-slate-800 border-rose-500/50 shadow-xl' : 'bg-slate-900/40 border-slate-800 hover:border-slate-700'
                            }`}
                        >
                            <div className="flex justify-between items-start mb-2">
                                <h3 className="font-bold text-white group-hover:text-rose-400 transition-colors uppercase tracking-wider text-sm">{role.name}</h3>
                                <Shield className={`w-4 h-4 ${selectedRole?.id === role.id ? 'text-rose-500' : 'text-slate-600'}`} />
                            </div>
                            <p className="text-xs text-slate-500 line-clamp-2">{role.description}</p>
                            <div className="mt-4 flex items-center gap-2 text-[10px] font-bold text-slate-400">
                                <span className="bg-slate-950 px-2 py-0.5 rounded">{t('roleManager.permissions_count', { count: role.permissions?.length || 0 })}</span>
                            </div>
                        </div>
                    ))}
                </div>

                {/* Permissions Editor */}
                <div className="lg:col-span-2">
                    {isEditing ? (
                        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 space-y-8 sticky top-8 animate-in slide-in-from-right-4">
                            <div className="flex justify-between items-center border-b border-slate-800 pb-4">
                                <h3 className="text-xl font-bold text-white">{selectedRole ? t('roleManager.edit_role') : t('roleManager.new_role')}</h3>
                                <button onClick={() => setIsEditing(false)} className="text-slate-500 hover:text-white transition-colors"><X className="w-5 h-5" /></button>
                            </div>

                            <div className="grid gap-6">
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('roleManager.role_name')}</label>
                                    <input 
                                        type="text"
                                        value={formData.name}
                                        onChange={e => setFormData({ ...formData, name: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500 transition-colors"
                                        placeholder={t('roleManager.role_name_placeholder')}
                                    />
                                </div>
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">{t('roleManager.description')}</label>
                                    <textarea 
                                        value={formData.description}
                                        onChange={e => setFormData({ ...formData, description: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500 transition-colors resize-none"
                                        rows="2"
                                        placeholder={t('roleManager.description_placeholder')}
                                    />
                                </div>
                            </div>

                            <div>
                                <label className="text-xs font-bold text-slate-500 uppercase block mb-4 flex items-center gap-2">
                                    <CheckSquare className="w-4 h-4 text-rose-500" />
                                    {t('roleManager.assign_permissions')}
                                </label>
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-[400px] overflow-auto pr-2 custom-scrollbar">
                                    {permissions.map(p => (
                                        <button 
                                            key={p.id}
                                            onClick={() => handleTogglePermission(p.name)}
                                            className={`p-3 rounded-xl border text-left flex items-start gap-3 transition-all ${
                                                formData.permissions.includes(p.name)
                                                    ? 'bg-rose-500/10 border-rose-500/30 text-rose-400'
                                                    : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'
                                            }`}
                                        >
                                            <div className="mt-0.5">
                                                {formData.permissions.includes(p.name) ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4" />}
                                            </div>
                                            <div>
                                                <div className="text-xs font-bold uppercase tracking-tighter">{p.name.replace('_', ' ')}</div>
                                                <div className="text-[10px] opacity-60 leading-tight mt-0.5">{p.description}</div>
                                            </div>
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <button 
                                onClick={handleSave}
                                className="w-full py-4 bg-rose-600 hover:bg-rose-500 text-white rounded-2xl font-bold flex items-center justify-center gap-2 transition-all shadow-xl shadow-rose-900/20"
                            >
                                <Save className="w-5 h-5" /> {t('roleManager.save_role_config')}
                            </button>
                        </div>
                    ) : (
                        <div className="bg-slate-950/20 border-2 border-dashed border-slate-900 rounded-3xl h-full flex flex-col items-center justify-center p-12 text-center text-slate-600">
                            <Shield className="w-16 h-16 mb-6 opacity-10" />
                            <h3 className="text-lg font-bold mb-2">{t('roleManager.no_role_selected')}</h3>
                            <p className="text-sm max-w-xs">{t('roleManager.no_role_selected_hint')}</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
