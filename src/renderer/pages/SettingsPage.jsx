// ========================================
// 設定頁面 (SettingsPage)
// 使用 shadcn Resizable, Collapsible 打造 VS Code 風格
// 目錄結構與設定項目從 settingsConfig.js 動態生成，無需手動維護 SETTINGS_TREE
// ========================================

import { useState, useRef } from 'react'
import { ChevronRight, RotateCcw } from 'lucide-react'
import useSettingsStore from '../stores/useSettingsStore'
import { SETTINGS_CONFIG, buildSettingsTree } from '../config/settingsConfig'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '@/components/ui/resizable'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

// 從 settingsConfig.js 自動組合目錄樹（不再手動維護 SETTINGS_TREE）
const SETTINGS_TREE = buildSettingsTree()

// ========================================
// SettingControl — 根據設定類型動態渲染對應的控制元件
// ========================================

/**
 * 根據 settingsConfig 中的 type 動態渲染設定控制元件。
 *
 * @param {Object} props
 * @param {Object} props.config - settingsConfig 中的設定項目定義
 * @param {any} props.value - 目前的設定值（來自 Store）
 * @param {Function} props.onChange - 值變更的回呼函數
 * @param {Array} [props.dynamicOptions] - 動態選項（當 config.options 為 null 時使用）
 * @returns {JSX.Element|null}
 */
function SettingControl({ config, value, onChange, dynamicOptions }) {
    switch (config.type) {
        case 'toggle':
            return (
                <Switch
                    checked={value === true || value === 'dark'}
                    onCheckedChange={(checked) => {
                        // toggle 類型特殊處理：theme 用 'dark'/'light'，其他用 boolean
                        if (config.key === 'theme') {
                            onChange(checked ? 'dark' : 'light')
                        } else {
                            onChange(checked)
                        }
                    }}
                />
            )
        case 'select': {
            const opts = config.options ?? dynamicOptions ?? []
            return (
                <Select value={value} onValueChange={onChange} disabled={opts.length === 0}>
                    <SelectTrigger className="w-64">
                        <SelectValue placeholder="請選擇" />
                    </SelectTrigger>
                    <SelectContent>
                        {opts.map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            )
        }
        default:
            return null
    }
}

// ========================================
// 設定頁面主元件
// ========================================

/**
 * 設定頁面元件。
 *
 * 提供 VS Code 風格的設定介面：左側目錄（可收合）、右側動態設定列表。
 * 目錄結構與設定項目均由 settingsConfig.js 驅動，無需手動維護。
 * 右上角提供「還原預設值」按鈕（含 AlertDialog 二次確認）。
 *
 * @returns {JSX.Element}
 */
function SettingsPage() {
    // 從 Store 取出所有設定值與方法
    const store = useSettingsStore()
    const {
        theme, toggleTheme,
        setThemeId,
        availableThemes,
        resetToDefaults,
    } = store

    // 展開狀態（預設全展開）
    const [openCategories, setOpenCategories] = useState(
        () => SETTINGS_TREE.reduce((acc, cat) => ({ ...acc, [cat.id]: true }), {})
    )

    // 目前高亮的設定區塊
    const [activeSection, setActiveSection] = useState(
        SETTINGS_TREE[0]?.items[0]?.id ?? ''
    )

    // 右側捲動容器 ref
    const rightScrollRef = useRef(null)
    // 各設定區塊的 DOM ref
    const sectionRefs = useRef({})

    const toggleCategory = (catId) => {
        setOpenCategories(prev => ({ ...prev, [catId]: !prev[catId] }))
    }

    /**
     * 捲動至右側指定的設定區塊。
     *
     * @param {string} id - 目標子分類 ID
     */
    const scrollToSection = (id) => {
        setActiveSection(id)
        const target = sectionRefs.current[id]
        const container = rightScrollRef.current
        if (target && container) {
            const containerRect = container.getBoundingClientRect()
            const targetRect = target.getBoundingClientRect()
            const offset = targetRect.top - containerRect.top + container.scrollTop - 24
            container.scrollTo({ top: offset, behavior: 'smooth' })
        }
    }

    /**
     * 根據設定的 key，從 Store 取出對應的 onChange handler。
     * theme / activeThemeId 有各自的 action；其他通用設定透過 updateSettings。
     *
     * @param {string} key - settingsConfig 中的設定 key
     * @returns {Function} onChange 回呼
     */
    const getChangeHandler = (key) => {
        switch (key) {
            case 'theme': return (val) => {
                // 切換主題時呼叫 toggleTheme（其內部根據新值決定 dark/light）
                if ((val === 'dark') !== (theme === 'dark')) toggleTheme()
            }
            case 'activeThemeId': return setThemeId
            default: return (val) => store.updateSettings({ [key]: val })
        }
    }

    // 動態選項提供者（當 config.options 為 null 時使用）
    const getDynamicOptions = (key) => {
        if (key === 'activeThemeId') {
            return availableThemes.map(t => ({ value: t.id, label: t.name }))
        }
        return []
    }

    return (
        <div className="h-full w-full bg-background animate-fade-in flex flex-col pt-2">

            {/* 頁面頂部標題 */}
            <div className="px-6 py-4 shrink-0 flex items-center justify-between border-b border-border">
                <div>
                    <h2 className="text-xl font-bold tracking-tight text-foreground">設定 (Settings)</h2>
                    <p className="text-sm text-muted-foreground mt-1">管理應用程式環境與偏好</p>
                </div>

                {/* 右上角：還原預設值按鈕（含二次確認） */}
                <AlertDialog>
                    <AlertDialogTrigger asChild>
                        <Button variant="outline" size="sm" className="gap-2 text-muted-foreground hover:text-foreground">
                            <RotateCcw size={14} />
                            還原預設值
                        </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>確認還原所有設定？</AlertDialogTitle>
                            <AlertDialogDescription>
                                此操作將把所有設定還原至出廠預設值，包括主題、色彩等。此動作無法復原。
                            </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>取消</AlertDialogCancel>
                            <AlertDialogAction onClick={resetToDefaults}>確認還原</AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </div>

            <div className="flex-1 overflow-hidden">
                {/* react-resizable-panels v4：size 需傳百分比字串 */}
                <ResizablePanelGroup direction="horizontal" className="h-full items-stretch">

                    {/* ========== 左側目錄（動態生成） ========== */}
                    <ResizablePanel defaultSize="25%" minSize="20%" maxSize="40%" className="bg-muted/10 flex flex-col overflow-hidden">
                        <div className="flex-1 overflow-y-auto py-4 pr-3 pl-2">
                            <div className="space-y-1">
                                {SETTINGS_TREE.map((category) => {
                                    const isOpen = openCategories[category.id]
                                    const Icon = category.icon
                                    return (
                                        <Collapsible
                                            key={category.id}
                                            open={isOpen}
                                            onOpenChange={() => toggleCategory(category.id)}
                                            className="w-full"
                                        >
                                            <CollapsibleTrigger className="flex items-center w-full py-1.5 px-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground rounded-md transition-colors [&[data-state=open]>svg.chevron]:rotate-90">
                                                <ChevronRight className="chevron h-4 w-4 shrink-0 transition-transform duration-200 text-muted-foreground mr-1" />
                                                <Icon size={16} className="text-muted-foreground mr-1.5" />
                                                {category.title}
                                            </CollapsibleTrigger>
                                            <CollapsibleContent className="px-3 pt-1 pb-2 space-y-0.5">
                                                {category.items.map(item => (
                                                    <div
                                                        key={item.id}
                                                        onClick={() => scrollToSection(item.id)}
                                                        className={`cursor-pointer px-6 py-1.5 text-sm rounded-md transition-colors ${
                                                            activeSection === item.id
                                                                ? 'bg-primary/10 text-primary font-medium'
                                                                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
                                                        }`}
                                                    >
                                                        {item.title}
                                                    </div>
                                                ))}
                                            </CollapsibleContent>
                                        </Collapsible>
                                    )
                                })}
                            </div>
                        </div>
                    </ResizablePanel>

                    <ResizableHandle withHandle className="hover:bg-primary/50 transition-colors" />

                    {/* ========== 右側設定內容（動態渲染） ========== */}
                    <ResizablePanel defaultSize="75%" className="bg-background flex flex-col overflow-hidden">
                        <div ref={rightScrollRef} className="flex-1 overflow-y-auto px-8 py-6">
                            <div className="max-w-3xl mx-auto pb-20">
                                {SETTINGS_TREE.map((category, catIndex) => (
                                    <div key={category.id}>
                                        {catIndex > 0 && <Separator className="my-10" />}

                                        {category.items.map((subCat, subIndex) => {
                                            // 找出此子分類下的所有設定項目
                                            const itemSettings = SETTINGS_CONFIG.filter(
                                                cfg => cfg.subCategoryId === subCat.id
                                            )
                                            return (
                                                <section
                                                    key={subCat.id}
                                                    ref={el => sectionRefs.current[subCat.id] = el}
                                                    className={`scroll-mt-6 ${subIndex > 0 ? 'mt-10' : ''}`}
                                                >
                                                    {/* 子分類標題 */}
                                                    <div className="mb-5">
                                                        <h3 className="text-base font-semibold text-foreground">
                                                            {subCat.title}
                                                        </h3>
                                                        <p className="text-xs text-muted-foreground mt-0.5">
                                                            {category.title} &rsaquo; {subCat.title}
                                                        </p>
                                                    </div>

                                                    {/* 設定項目列表 */}
                                                    <div className="space-y-6">
                                                        {itemSettings.map(cfg => (
                                                            <div key={cfg.key} className="flex items-start justify-between gap-8 max-w-2xl">
                                                                <div className="flex flex-col gap-1 flex-1 min-w-0">
                                                                    <Label className="text-sm font-normal cursor-pointer">
                                                                        {cfg.title}
                                                                    </Label>
                                                                    <p className="text-xs text-muted-foreground">
                                                                        {cfg.description}
                                                                    </p>
                                                                </div>
                                                                <div className="shrink-0 flex items-center min-h-[36px]">
                                                                    <SettingControl
                                                                        config={cfg}
                                                                        value={store[cfg.key]}
                                                                        onChange={getChangeHandler(cfg.key)}
                                                                        dynamicOptions={getDynamicOptions(cfg.key)}
                                                                    />
                                                                </div>
                                                            </div>
                                                        ))}
                                                    </div>
                                                </section>
                                            )
                                        })}
                                    </div>
                                ))}
                            </div>
                        </div>
                    </ResizablePanel>

                </ResizablePanelGroup>
            </div>
        </div>
    )
}

export default SettingsPage
