import React, { useState, useRef } from 'react';
import { FileSpreadsheet, Upload, AlertCircle, CheckCircle, X, ChevronRight, Settings } from 'lucide-react';
import * as XLSX from 'xlsx';
import apiClient from '../api/axios';
import { useTranslation } from 'react-i18next';

const ExcelImport = () => {
    const { t } = useTranslation();
    const [dragActive, setDragActive] = useState(false);
    const [file, setFile] = useState(null);
    const [uploading, setUploading] = useState(false);
    const [uploadStatus, setUploadStatus] = useState(null); // success, error
    const inputRef = useRef(null);

    // Excel Parsing State
    const [excelData, setExcelData] = useState([]);
    const [columns, setColumns] = useState([]);
    const [mapping, setMapping] = useState({ phone: '', message: '' });
    const [isProcessing, setIsProcessing] = useState(false);

    const handleDrag = (e) => {
        e.preventDefault();
        e.stopPropagation();
        if (e.type === "dragenter" || e.type === "dragover") {
            setDragActive(true);
        } else if (e.type === "dragleave") {
            setDragActive(false);
        }
    };

    const handleDrop = (e) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
        if (e.dataTransfer.files && e.dataTransfer.files[0]) {
            processFile(e.dataTransfer.files[0]);
        }
    };

    const handleChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            processFile(e.target.files[0]);
        }
    };

    const processFile = (selectedFile) => {
        setFile(selectedFile);
        setIsProcessing(true);
        setUploadStatus(null);
        setExcelData([]);
        setColumns([]);
        setMapping({ phone: '', message: '' });

        const reader = new FileReader();
        reader.onload = (e) => {
            try {
                const data = new Uint8Array(e.target.result);
                const workbook = XLSX.read(data, { type: 'array' });
                const firstSheetName = workbook.SheetNames[0];
                const worksheet = workbook.Sheets[firstSheetName];

                // Read as JSON to get columns and data
                const jsonData = XLSX.utils.sheet_to_json(worksheet, { header: 1 });

                if (jsonData.length > 0) {
                    const headers = jsonData[0].map((h, i) => h ? String(h) : `Column ${i + 1}`);
                    setColumns(headers);

                    // Keep header and format remaining rows as objects
                    const rowsAsObjects = XLSX.utils.sheet_to_json(worksheet);
                    setExcelData(rowsAsObjects);

                    // Auto-guess columns
                    let guessPhone = '';
                    let guessMsg = '';
                    headers.forEach(h => {
                        const lowerH = h.toLowerCase();
                        if (lowerH.includes('phone') || lowerH.includes('رقم') || lowerH.includes('هاتف')) guessPhone = h;
                        if (lowerH.includes('message') || lowerH.includes('رسالة') || lowerH.includes('نص')) guessMsg = h;
                    });
                    setMapping({ phone: guessPhone, message: guessMsg });
                }
            } catch (error) {
                console.error("Error reading file:", error);
                setUploadStatus('error');
            } finally {
                setIsProcessing(false);
            }
        };
        reader.readAsArrayBuffer(selectedFile);
    };

    const handleUpload = async () => {
        if (!file || !mapping.phone || !mapping.message || excelData.length === 0) return;
        setUploading(true);
        setUploadStatus(null);

        // Create the Ingestion Payload for the bulk Excel endpoint
        const ingestionPayload = {
            sender_name: "Web-Excel-Import",
            message_type: "",
            messages: excelData.map(row => ({
                phone_number: row[mapping.phone] ? String(row[mapping.phone]) : '',
                content: row[mapping.message] ? String(row[mapping.message]) : '',
            })).filter(item => item.phone_number && item.content)
        };

        if (ingestionPayload.messages.length === 0) {
            setUploadStatus('error');
            setUploading(false);
            alert(t('excelImport.no_valid_rows'));
            return;
        }

        try {
            await apiClient.post('/api/v1/messaging/excel-ingest', ingestionPayload);
            setUploadStatus('success');
        } catch (error) {
            console.error(error);
            setUploadStatus('error');
        } finally {
            setUploading(false);
        }
    };

    const resetImport = () => {
        setFile(null);
        setExcelData([]);
        setColumns([]);
        setMapping({ phone: '', message: '' });
        setUploadStatus(null);
    };

    return (
        <div className="animate-in fade-in zoom-in-95 duration-500">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h2 className="text-3xl font-bold">إرسال عبر ملف إكسل</h2>
                    <p className="text-slate-400 text-sm mt-1">قم برفع ملف يحتوي على الأرقام والرسائل ليتم جدولتها للإرسال.</p>
                </div>
            </div>

            {!file ? (
                <div
                    className={`glass-card p-16 text-center border-dashed border-2 transition-all group cursor-pointer
                        ${dragActive ? 'border-rose-500 bg-rose-500/5' : 'border-slate-700 hover:border-rose-500/50'}
                    `}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                    onClick={() => inputRef.current?.click()}
                >
                    <input
                        ref={inputRef}
                        type="file"
                        className="hidden"
                        accept=".xlsx,.xls,.csv"
                        onChange={handleChange}
                    />
                    <div className="bg-slate-800 w-24 h-24 rounded-3xl flex items-center justify-center mx-auto mb-6 group-hover:scale-110 transition-transform shadow-xl">
                        <FileSpreadsheet className="w-12 h-12 text-rose-500" />
                    </div>
                    <h3 className="text-2xl font-bold mb-3">اسحب وأفلت الملف هنا</h3>
                    <p className="text-slate-500 mb-8 max-w-sm mx-auto leading-relaxed">يدعم صيغ .xlsx و .xls و .csv بأقصى حد 50,000 صف للملف الواحد.</p>
                    <button className="btn-premium px-8 py-3 w-auto inline-block">تصفح الملفات</button>
                </div>
            ) : (
                <div className="space-y-6">
                    {/* Header Info */}
                    <div className="glass-card p-6 flex flex-col md:flex-row md:items-center justify-between gap-6 border-l-4 border-l-rose-500">
                        <div className="flex items-center gap-4">
                            <div className="bg-slate-800 w-16 h-16 rounded-2xl flex items-center justify-center shrink-0">
                                <FileSpreadsheet className="w-8 h-8 text-emerald-500" />
                            </div>
                            <div>
                                <h3 className="text-xl font-bold mb-1 truncate max-w-sm" title={file.name}>{file.name}</h3>
                                <div className="flex items-center gap-3 text-sm text-slate-400 font-mono">
                                    <span>{(file.size / 1024).toFixed(2)} KB</span>
                                    <span className="w-1 h-1 bg-slate-700 rounded-full" />
                                    <span>{excelData.length} صفوف</span>
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-3">
                            <button onClick={resetImport} className="px-5 py-2.5 rounded-xl border border-slate-700 hover:bg-slate-800 transition-colors text-sm font-bold">
                                إلغاء الملف
                            </button>
                        </div>
                    </div>

                    {/* Column Mapping */}
                    {columns.length > 0 && uploadStatus !== 'success' && (
                        <div className="glass-card p-8">
                            <div className="flex items-center gap-3 mb-6">
                                <Settings className="w-5 h-5 text-rose-500" />
                                <h3 className="text-lg font-bold">{t('excelImport.column_mapping')}</h3>
                            </div>

                            <div className="grid md:grid-cols-2 gap-8">
                                <div>
                                    <label className="block text-sm font-bold text-slate-300 mb-2">عمود رقم الهاتف <span className="text-rose-500">*</span></label>
                                    <select
                                        value={mapping.phone}
                                        onChange={(e) => setMapping({ ...mapping, phone: e.target.value })}
                                        className="w-full bg-slate-900 border border-slate-700 rounded-xl py-3 px-4 focus:outline-none focus:border-rose-500 transition-colors"
                                    >
                                        <option value="">-- اختر العمود --</option>
                                        {columns.map(c => <option key={`p-${c}`} value={c}>{c}</option>)}
                                    </select>
                                    <p className="text-xs text-slate-500 mt-2">اختر العمود الذي يحتوي على أرقام هواتف المستلمين.</p>
                                </div>
                                <div>
                                    <label className="block text-sm font-bold text-slate-300 mb-2">عمود نص الرسالة <span className="text-rose-500">*</span></label>
                                    <select
                                        value={mapping.message}
                                        onChange={(e) => setMapping({ ...mapping, message: e.target.value })}
                                        className="w-full bg-slate-900 border border-slate-700 rounded-xl py-3 px-4 focus:outline-none focus:border-rose-500 transition-colors"
                                    >
                                        <option value="">-- اختر العمود --</option>
                                        {columns.map(c => <option key={`m-${c}`} value={c}>{c}</option>)}
                                    </select>
                                    <p className="text-xs text-slate-500 mt-2">اختر العمود الذي يحتوي على نصوص الرسائل المراد إرسالها.</p>
                                </div>
                            </div>

                            <div className="mt-8 pt-6 border-t border-slate-800 flex justify-end">
                                <button
                                    onClick={handleUpload}
                                    disabled={uploading || !mapping.phone || !mapping.message}
                                    className="btn-premium flex items-center gap-2 px-8 py-3 disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {uploading ? (
                                        <>
                                            <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                                            جاري الإرسال...
                                        </>
                                    ) : (
                                        <>
                                            اعتماد وإرسال البيانات
                                            <Upload className="w-4 h-4" />
                                        </>
                                    )}
                                </button>
                            </div>

                            {uploadStatus === 'error' && (
                                <div className="mt-4 p-4 bg-rose-500/10 border border-rose-500/20 rounded-xl flex items-center justify-center gap-2 text-rose-400">
                                    <X className="w-5 h-5" /> حدث خطأ أثناء إرسال البيانات. تأكد من صحة الأعمدة والاتصال بالخادم.
                                </div>
                            )}
                        </div>
                    )}

                    {uploadStatus === 'success' && (
                        <div className="p-8 bg-emerald-500/10 border border-emerald-500/20 rounded-2xl flex flex-col items-center justify-center text-center">
                            <div className="w-16 h-16 bg-emerald-500/20 rounded-full flex items-center justify-center mb-4">
                                <CheckCircle className="w-8 h-8 text-emerald-400" />
                            </div>
                            <h3 className="text-2xl font-bold text-emerald-400 mb-2">تم استلام البيانات بنجاح!</h3>
                            <p className="text-emerald-500/70 mb-6 max-w-md">تم رفع {excelData.length} سجل وجدولتها للإرسال عبر أجهزة الهاتف المتاحة.</p>
                            <button onClick={resetImport} className="px-6 py-2 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 rounded-xl font-bold transition-colors">
                                رفع ملف آخر
                            </button>
                        </div>
                    )}

                    {/* Data Preview */}
                    {excelData.length > 0 && uploadStatus !== 'success' && (
                        <div className="glass-card overflow-hidden">
                            <div className="p-6 border-b border-slate-800/50 flex items-center justify-between">
                                <div>
                                    <h3 className="font-bold text-lg">معاينة البيانات</h3>
                                    <p className="text-xs text-slate-500">نعرض أول 5 صفوف فقط من الملف للتأكد من صحة القراءة.</p>
                                </div>
                            </div>
                            <div className="overflow-x-auto">
                                <table className="w-full text-left whitespace-nowrap">
                                    <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-black">
                                        <tr>
                                            <th className="px-6 py-4 w-12 text-center">#</th>
                                            {columns.map(col => (
                                                <th key={col} className={`px-6 py-4 ${mapping.phone === col ? 'text-blue-400 bg-blue-500/5' : ''} ${mapping.message === col ? 'text-purple-400 bg-purple-500/5' : ''}`}>
                                                    {col}
                                                    {mapping.phone === col && <span className="ml-2 inline-block px-2 py-0.5 bg-blue-500/20 rounded text-[9px] text-blue-300">{t('excelImport.phone_badge')}</span>}
                                                    {mapping.message === col && <span className="ml-2 inline-block px-2 py-0.5 bg-purple-500/20 rounded text-[9px] text-purple-300">{t('excelImport.message_badge')}</span>}
                                                </th>
                                            ))}
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-800">
                                        {excelData.slice(0, 5).map((row, idx) => (
                                            <tr key={idx} className="hover:bg-slate-800/30 transition-colors">
                                                <td className="px-6 py-4 text-center text-xs text-slate-600 font-mono">{idx + 1}</td>
                                                {columns.map(col => (
                                                    <td key={`${idx}-${col}`} className={`px-6 py-4 text-sm text-slate-300 truncate max-w-xs ${mapping.phone === col || mapping.message === col ? 'font-bold' : 'text-slate-500'}`}>
                                                        {row[col] !== undefined && row[col] !== null ? String(row[col]) : <span className="text-slate-700 italic">فارغ</span>}
                                                    </td>
                                                ))}
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                            {excelData.length > 5 && (
                                <div className="p-4 text-center border-t border-slate-800/50 bg-slate-900/30 text-xs text-slate-500">
                                    + {excelData.length - 5} صفوف إضافية مخفية في المعاينة
                                </div>
                            )}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};

export default ExcelImport;
