// ========================================
// 設定頁面 (SettingsPage)
// 使用 shadcn Resizable, ScrollArea, Collapsible 打造 VS Code 風格
// ========================================

import { useState, useRef } from 'react'
import { Moon, Sun, Palette, ChevronRight, Monitor, Settings2 } from 'lucide-react'
import useSettingsStore from '../stores/useSettingsStore'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

// 定義設定目錄結構與對應的標示
const SETTINGS_TREE = [
    {
        id: 'general',
        title: '一般',
        icon: <Settings2 size={16} className="text-muted-foreground mr-1.5" />,
        items: [
            { id: 'general.startup', title: '啟動行為' }
        ]
    },
    {
        id: 'appearance',
        title: '外觀',
        icon: <Monitor size={16} className="text-muted-foreground mr-1.5" />,
        items: [
            { id: 'appearance.theme', title: '佈景主題' },
            { id: 'appearance.colors', title: '主題色彩' }
        ]
    }
]

/**
 * 設定頁面元件。
 *
 * 提供 Dark/Light 主題切換、配色風格選擇。
 * 以左右兩側 Resizable Panel 呈現，左側為目錄，右側為設定選項。
 *
 * @returns {JSX.Element}
 */
function SettingsPage() {
    const { theme, toggleTheme, activeThemeId, availableThemes, setThemeId } = useSettingsStore()
    
    // 用於記錄各種分類展開狀態
    const [openCategories, setOpenCategories] = useState({
        general: true,
        appearance: true
    })

    // 用於記錄目前選擇或捲動到的設定項目
    const [activeSection, setActiveSection] = useState('general.startup')

    // 右側內容區的 ScrollArea viewport ref（用於強制從上方捲動）
    const rightScrollRef = useRef(null)
    // 設定各區塊的 ref（注意：存的是 DOM 元素，不是 ScrollArea 元素）
    const sectionRefs = useRef({})

    const toggleCategory = (catId) => {
        setOpenCategories(prev => ({ ...prev, [catId]: !prev[catId] }))
    }

    /**
     * 捲動至右側指定的設定區塊。
     * 取得目標元素相對於 ScrollArea viewport 的偏移量，再設定 scrollTop。
     * 
     * @param {string} id - 目標設定區塊的 ID
     */
    const scrollToSection = (id) => {
        setActiveSection(id)
        const target = sectionRefs.current[id]
        const container = rightScrollRef.current
        if (target && container) {
            // 計算目標元素相對於可滾動容器的頂部偏移量
            const containerRect = container.getBoundingClientRect()
            const targetRect = target.getBoundingClientRect()
            const offset = targetRect.top - containerRect.top + container.scrollTop - 24
            container.scrollTo({ top: offset, behavior: 'smooth' })
        }
    }

    return (
        <div className="h-full w-full bg-background animate-fade-in flex flex-col pt-2">
            
            {/* 頁面頂部標題（仿 VSCode search列附近） */}
            <div className="px-6 py-4 shrink-0 flex items-center justify-between border-b border-border">
                <div>
                    <h2 className="text-xl font-bold tracking-tight text-foreground">設定 (Settings)</h2>
                    <p className="text-sm text-muted-foreground mt-1">管理應用程式環境與偏好</p>
                </div>
            </div>

            <div className="flex-1 overflow-hidden">
                {/* react-resizable-panels v4 重要：defaultSize/minSize/maxSize 需用百分比字串（如 "25%"），傳數字時會被解讀為像素！ */}
                <ResizablePanelGroup direction="horizontal" className="h-full items-stretch">
                    
                    {/* ========== 左側目錄 ========== */}
                    <ResizablePanel defaultSize="25%" minSize="20%" maxSize="40%" className="bg-muted/10 flex flex-col overflow-hidden">
                        <ScrollArea className="flex-1 py-4 pr-3 pl-2">
                            <div className="space-y-1">
                                {SETTINGS_TREE.map((category) => {
                                    const isOpen = openCategories[category.id]
                                    return (
                                        <Collapsible 
                                            key={category.id} 
                                            open={isOpen} 
                                            onOpenChange={() => toggleCategory(category.id)}
                                            className="w-full"
                                        >
                                            <CollapsibleTrigger className="flex items-center w-full py-1.5 px-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground rounded-md transition-colors [&[data-state=open]>svg.chevron]:rotate-90">
                                                <ChevronRight className="chevron h-4 w-4 shrink-0 transition-transform duration-200 text-muted-foreground mr-1" />
                                                {category.icon}
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
                        </ScrollArea>
                    </ResizablePanel>

                    <ResizableHandle withHandle className="hover:bg-primary/50 transition-colors" />

                    {/* ========== 右側設定內容 ========== */}
                    <ResizablePanel defaultSize="75%" className="bg-background flex flex-col overflow-hidden">
                        {/* overflow-y-auto 加 ref 讓 scrollToSection 能正確定位 */}
                        <div ref={rightScrollRef} className="flex-1 overflow-y-auto px-8 py-6">
                            <div className="max-w-3xl mx-auto space-y-12 pb-20">
                                
                                {/* ---- 區塊：一般 / 啟動行為 ---- */}
                                <section ref={el => sectionRefs.current['general.startup'] = el} className="scroll-mt-6">
                                    <div className="mb-4">
                                        <h3 className="text-lg font-medium text-foreground">啟動行為 (Startup)</h3>
                                        <p className="text-sm text-muted-foreground">應用程式啟動時的相關設定</p>
                                    </div>
                                    <div className="space-y-6">
                                        {/* 假設定範例 */}
                                        <div className="flex flex-col gap-2">
                                            <Label className="text-base font-normal">啟動時開啟的頁面</Label>
                                            <p className="text-sm text-muted-foreground">
                                                控制啟動 BOMIX 時預設導航的頁面。
                                            </p>
                                            <Select defaultValue="dashboard" disabled>
                                                <SelectTrigger className="w-64 mt-1">
                                                    <SelectValue placeholder="選擇預設頁面" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="dashboard">Dashboard (預設)</SelectItem>
                                                    <SelectItem value="bom">BOM 編輯器</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </div>
                                    </div>
                                </section>
                                
                                <Separator />

                                {/* ---- 區塊：外觀 / 佈景主題 ---- */}
                                <section ref={el => sectionRefs.current['appearance.theme'] = el} className="scroll-mt-6">
                                    <div className="mb-4 flex items-center gap-2 text-foreground">
                                        <h3 className="text-lg font-medium">佈景主題 (Theme)</h3>
                                    </div>
                                    <div className="space-y-6">
                                        <div className="flex items-center justify-between max-w-lg">
                                            <div className="flex flex-col gap-1.5">
                                                <Label className="text-base font-normal cursor-pointer flex items-center gap-2" htmlFor="theme-switch">
                                                    啟用深色模式
                                                    {theme === 'light' ? <Sun size={16} className="text-amber-500" /> : <Moon size={16} className="text-indigo-400" />}
                                                </Label>
                                                <p className="text-sm text-muted-foreground">
                                                    切換應用程式整體的明暗主題
                                                </p>
                                            </div>
                                            <Switch
                                                id="theme-switch"
                                                checked={theme === 'dark'}
                                                onCheckedChange={toggleTheme}
                                            />
                                        </div>
                                    </div>
                                </section>

                                <Separator />
                                
                                {/* ---- 區塊：外觀 / 主題色彩 ---- */}
                                <section ref={el => sectionRefs.current['appearance.colors'] = el} className="scroll-mt-6">
                                    <div className="mb-4 flex items-center gap-2 text-foreground">
                                        <h3 className="text-lg font-medium">主題色彩 (Color Palette)</h3>
                                    </div>
                                    <div className="space-y-6">
                                        <div className="flex flex-col gap-2">
                                            <Label className="text-base font-normal flex items-center gap-2">
                                                顏色風格
                                                <Palette size={16} className="text-pink-500" />
                                            </Label>
                                            <p className="text-sm text-muted-foreground">
                                                選擇主要的品牌色彩與強調顏色
                                            </p>
                                            <Select value={activeThemeId} onValueChange={setThemeId}>
                                                <SelectTrigger className="w-64 mt-1">
                                                    <SelectValue placeholder="選擇主題" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {availableThemes.map(t => (
                                                        <SelectItem key={t.id} value={t.id}>
                                                            {t.name}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>
                                    </div>
                                </section>

                            </div>
                        </div>
                    </ResizablePanel>

                </ResizablePanelGroup>
            </div>
        </div>
    )
}

export default SettingsPage
