import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Building2 } from 'lucide-react'

function Dashboard() {
  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center p-8">
      <div className="text-center max-w-lg">
        <div className="p-6 bg-rose-500/10 rounded-[2rem] inline-block mb-8 border border-rose-500/20">
          <Building2 className="w-16 h-16 text-rose-500" />
        </div>
        <h1 className="text-4xl font-black text-white mb-4">Company Portal</h1>
        <p className="text-slate-400 text-lg mb-8">
          مرحباً بك في بوابة الشركات - منصة إدارة خدمات الهدهد
        </p>
        <div className="flex gap-4 justify-center">
          <div className="px-6 py-3 bg-slate-900 rounded-xl border border-slate-800 text-sm text-slate-300">
            إحصائيات قادمة قريباً
          </div>
        </div>
      </div>
    </div>
  )
}

function App() {
  return (
    <Routes>
      <Route path="/" element={<Dashboard />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
