import { useState, useEffect } from 'react'
import Dialog from './Dialog'
import { Check, X, FileText, Calendar, Clock } from 'lucide-react'

/**
 * BOM 屬性編輯對話框
 * 
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {Object} props.bom - 要編輯的 BOM 物件
 * @param {string} props.projectCode - 專案代碼
 * @param {Function} props.onSave - 儲存回呼 (id, updates) => Promise
 */
function BomMetaDialog({ isOpen, onClose, bom, projectCode, onSave }) {
    // Editable State
    const [description, setDescription] = useState('')
    const [note, setNote] = useState('')
    const [suffix, setSuffix] = useState('')
    const [schematicVersion, setSchematicVersion] = useState('')
    const [pcbVersion, setPcbVersion] = useState('')
    const [pcaPn, setPcaPn] = useState('')

    // Read Only State (for display)
    const [bomDate, setBomDate] = useState('')
    const [mode, setMode] = useState('NPI')
    const [filename, setFilename] = useState('')
    const [createdAt, setCreatedAt] = useState('')

    const [isSaving, setIsSaving] = useState(false)
    const [error, setError] = useState('')

    useEffect(() => {
        if (isOpen && bom) {
            setDescription(bom.description || '')
            setNote(bom.note || '')
            setSuffix(bom.suffix || '')
            setSchematicVersion(bom.schematic_version || '')
            setPcbVersion(bom.pcb_version || '')
            setPcaPn(bom.pca_pn || '')

            setBomDate(bom.bom_date || '')
            setMode(bom.mode || 'NPI')
            setFilename(bom.filename || '')
            setCreatedAt(bom.created_at || '')
            
            setError('')
        }
    }, [isOpen, bom])

    const handleSave = async () => {
        if (!bom) return

        setIsSaving(true)
        setError('')
        try {
            await onSave(bom.id, {
                description,
                note,
                suffix,
                schematic_version: schematicVersion,
                pcb_version: pcbVersion,
                pca_pn: pcaPn
            })
            // onClose is handled by parent upon success usually, or we close here if parent waits.
            // Dashboard.jsx waits for updateRevision then closes.
        } catch (err) {
            setError(err.message || '儲存失敗')
        } finally {
            setIsSaving(false)
        }
    }

    // Title Format: TANGLED SI 0.3 - 1
    const displayTitle = bom 
        ? `${projectCode || ''} ${bom.phase_name} ${bom.version} ${bom.suffix ? '- ' + bom.suffix : ''}`
        : '編輯 BOM 屬性'

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title="編輯 BOM 屬性"
            className="max-w-2xl"
        >
            <div className="space-y-6">
                {/* Header Info */}
                <div className="bg-slate-50 dark:bg-surface-800 p-4 rounded-xl border border-slate-100 dark:border-slate-700">
                    <h3 className="text-lg font-bold text-primary-600 dark:text-primary-400 flex items-center gap-2">
                        <FileText size={20} />
                        {displayTitle}
                    </h3>
                </div>

                {/* Editable Fields */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {/* Suffix */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Suffix
                        </label>
                        <input
                            type="text"
                            value={suffix}
                            onChange={(e) => setSuffix(e.target.value)}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
                     {/* Schematic Version */}
                     <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Schematic Version
                        </label>
                        <input
                            type="text"
                            value={schematicVersion}
                            onChange={(e) => setSchematicVersion(e.target.value)}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
                    {/* PCB Version */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            PCB Version
                        </label>
                        <input
                            type="text"
                            value={pcbVersion}
                            onChange={(e) => setPcbVersion(e.target.value)}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
                    {/* PCA PN */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            PCA PN
                        </label>
                        <input
                            type="text"
                            value={pcaPn}
                            onChange={(e) => setPcaPn(e.target.value)}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
                    
                    {/* Description (Full Width) */}
                    <div className="md:col-span-3">
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Description
                        </label>
                        <input
                            type="text"
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>

                    {/* Note (Full Width) */}
                    <div className="md:col-span-3">
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Note
                        </label>
                        <textarea
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            rows={2}
                            className="w-full px-3 py-2 text-sm bg-white dark:bg-surface-900 border border-slate-300 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
                </div>

                {/* Read-only Info */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 bg-slate-50 dark:bg-surface-900/50 rounded-xl border border-slate-100 dark:border-slate-800 text-xs">
                    <div>
                        <span className="block text-slate-400 mb-1">Mode</span>
                        <span className="font-semibold text-slate-700 dark:text-slate-300">{mode}</span>
                    </div>
                    <div>
                        <span className="block text-slate-400 mb-1">BOM Date</span>
                        <div className="flex items-center gap-1 text-slate-700 dark:text-slate-300">
                           <Calendar size={12} />
                           <span>{bomDate}</span>
                        </div>
                    </div>
                     <div className="md:col-span-2">
                        <span className="block text-slate-400 mb-1">Source File</span>
                        <span className="text-slate-700 dark:text-slate-300 truncate block" title={filename}>{filename}</span>
                    </div>
                     <div>
                        <span className="block text-slate-400 mb-1">Created At</span>
                         <div className="flex items-center gap-1 text-slate-700 dark:text-slate-300">
                           <Clock size={12} />
                           <span>{createdAt?.split('T')[0]}</span>
                        </div>
                    </div>
                </div>

                {/* Error Update */}
                {error && (
                    <div className="text-sm text-red-500 flex items-center gap-1">
                        <X size={14} />
                        {error}
                    </div>
                )}

                {/* Buttons */}
                <div className="flex justify-end gap-2 pt-2 border-t border-slate-100 dark:border-slate-800">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-700 rounded-lg transition-colors"
                    >
                        取消
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={isSaving}
                        className="flex items-center gap-2 px-4 py-2 text-sm text-white bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors disabled:opacity-50"
                    >
                        {isSaving ? '儲存中...' : <><Check size={16} /> 儲存</>}
                    </button>
                </div>
            </div>
        </Dialog>
    )
}

export default BomMetaDialog
