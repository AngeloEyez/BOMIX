// ========================================
// BOM 屬性編輯對話框 (BomMetaDialog)
// 使用 shadcn Input, Label, Button 統一風格
// ========================================

import { useState, useEffect } from 'react'
import Dialog from './Dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Check, FileText, Calendar, Clock, AlertCircle } from 'lucide-react'

/**
 * BOM 屬性編輯對話框。
 *
 * 提供 Suffix、Schematic Version、PCB Version、PCA PN、Description、Note 的編輯，
 * Mode、BOM Date、Source File、Created At 為唯讀顯示。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {Object} props.bom - 要編輯的 BOM 物件
 * @param {string} props.projectCode - 專案代碼（顯示用）
 * @param {Function} props.onSave - 儲存回呼 (id, updates) => Promise
 * @returns {JSX.Element}
 */
function BomMetaDialog({ isOpen, onClose, bom, projectCode, onSave }) {
    // 可編輯欄位
    const [description, setDescription] = useState('')
    const [note, setNote] = useState('')
    const [suffix, setSuffix] = useState('')
    const [schematicVersion, setSchematicVersion] = useState('')
    const [pcbVersion, setPcbVersion] = useState('')
    const [pcaPn, setPcaPn] = useState('')

    // 唯讀欄位（顯示用）
    const [bomDate, setBomDate] = useState('')
    const [mode, setMode] = useState('NPI')
    const [filename, setFilename] = useState('')
    const [createdAt, setCreatedAt] = useState('')

    const [isSaving, setIsSaving] = useState(false)
    const [error, setError] = useState('')

    // 開啟對話框時同步 BOM 資料
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

    /**
     * 送出儲存請求，將可編輯欄位傳給父元件的 onSave。
     */
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
        } catch (err) {
            setError(err.message || '儲存失敗')
        } finally {
            setIsSaving(false)
        }
    }

    const displayTitle = bom
        ? `${projectCode || ''} ${bom.phase_name} ${bom.version}${bom.suffix ? ' - ' + bom.suffix : ''}`
        : '編輯 BOM 屬性'

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title="編輯 BOM 屬性" className="max-w-2xl">
            <div className="space-y-4">
                {/* BOM 識別標頭 */}
                <div className="flex items-center gap-2 px-3 py-2.5 bg-primary/5 rounded-lg border border-primary/20">
                    <FileText size={16} className="text-primary shrink-0" />
                    <span className="text-sm font-semibold text-primary truncate">{displayTitle}</span>
                </div>

                {/* 可編輯欄位 */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                    {/* Suffix */}
                    <div className="space-y-1">
                        <Label className="text-xs">Suffix</Label>
                        <Input className="h-8 text-xs" value={suffix} onChange={(e) => setSuffix(e.target.value)} />
                    </div>
                    {/* Schematic Version */}
                    <div className="space-y-1">
                        <Label className="text-xs">Schematic Version</Label>
                        <Input className="h-8 text-xs" value={schematicVersion} onChange={(e) => setSchematicVersion(e.target.value)} />
                    </div>
                    {/* PCB Version */}
                    <div className="space-y-1">
                        <Label className="text-xs">PCB Version</Label>
                        <Input className="h-8 text-xs" value={pcbVersion} onChange={(e) => setPcbVersion(e.target.value)} />
                    </div>
                    {/* PCA PN */}
                    <div className="space-y-1">
                        <Label className="text-xs">PCA PN</Label>
                        <Input className="h-8 text-xs" value={pcaPn} onChange={(e) => setPcaPn(e.target.value)} />
                    </div>
                    {/* Description (全寬) */}
                    <div className="md:col-span-3 space-y-1">
                        <Label className="text-xs">Description</Label>
                        <Input className="h-8 text-xs" value={description} onChange={(e) => setDescription(e.target.value)} />
                    </div>
                    {/* Note (全寬) */}
                    <div className="md:col-span-3 space-y-1">
                        <Label className="text-xs">Note</Label>
                        <textarea
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            rows={2}
                            className="w-full px-3 py-1.5 text-xs
                                bg-background border border-input rounded-md
                                text-foreground placeholder:text-muted-foreground
                                focus:outline-none focus:ring-1 focus:ring-ring
                                resize-none transition-colors selectable"
                        />
                    </div>
                </div>

                {/* 唯讀資訊欄 */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-3 bg-muted/40 rounded-lg border border-border text-xs">
                    <div>
                        <span className="block text-muted-foreground mb-0.5">Mode</span>
                        <span className="font-semibold text-foreground">{mode}</span>
                    </div>
                    <div>
                        <span className="block text-muted-foreground mb-0.5">BOM Date</span>
                        <div className="flex items-center gap-1 text-foreground">
                            <Calendar size={11} />
                            <span>{bomDate}</span>
                        </div>
                    </div>
                    <div className="md:col-span-2">
                        <span className="block text-muted-foreground mb-0.5">Source File</span>
                        <span className="text-foreground truncate block font-mono" title={filename}>{filename}</span>
                    </div>
                    <div>
                        <span className="block text-muted-foreground mb-0.5">Created At</span>
                        <div className="flex items-center gap-1 text-foreground">
                            <Clock size={11} />
                            <span>{createdAt?.split('T')[0]}</span>
                        </div>
                    </div>
                </div>

                {/* 錯誤訊息 */}
                {error && (
                    <div className="flex items-center gap-1.5 text-xs text-destructive">
                        <AlertCircle size={13} />
                        {error}
                    </div>
                )}

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2 pt-1 border-t border-border">
                    <Button variant="outline" size="sm" onClick={onClose}>取消</Button>
                    <Button size="sm" onClick={handleSave} disabled={isSaving}>
                        {isSaving ? '儲存中...' : <><Check size={14} className="mr-1" />儲存</>}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default BomMetaDialog
