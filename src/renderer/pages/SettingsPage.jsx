// ========================================
// 設定頁面 (SettingsPage)
// 使用 shadcn Card, Switch, Select 統一風格
// ========================================

import { useState } from 'react'
import { Info, FileText, Moon, Sun, Palette } from 'lucide-react'
import useSettingsStore from '../stores/useSettingsStore'
import AboutDialog from '../components/dialogs/AboutDialog'
import ChangelogDialog from '../components/dialogs/ChangelogDialog'
import { Card, CardContent } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'

/**
 * 設定頁面元件。
 *
 * 提供 Dark/Light 主題切換、配色風格選擇，
 * 以及關於對話框與更新記錄的進入點。
 *
 * @returns {JSX.Element}
 */
function SettingsPage() {
    const { theme, toggleTheme, activeThemeId, availableThemes, setThemeId } = useSettingsStore()
    const [isAboutOpen, setIsAboutOpen] = useState(false)
    const [isChangelogOpen, setIsChangelogOpen] = useState(false)

    return (
        <div className="max-w-xl mx-auto py-6 px-4 space-y-6 animate-fade-in">
            {/* 頁面標題 */}
            <div>
                <h2 className="text-base font-semibold text-foreground">設定</h2>
                <p className="text-xs text-muted-foreground mt-0.5">管理應用程式偏好設定與查看資訊。</p>
            </div>

            {/* ==================== 一般設定 ==================== */}
            <section className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">一般</h3>

                <Card>
                    <CardContent className="p-0 divide-y divide-border">
                        {/* Dark/Light 模式切換 */}
                        <div className="flex items-center justify-between px-4 py-3">
                            <div className="flex items-center gap-3">
                                <div className="p-1.5 bg-primary/10 text-primary rounded-md">
                                    {theme === 'light' ? <Sun size={16} /> : <Moon size={16} />}
                                </div>
                                <div>
                                    <Label className="text-sm font-medium cursor-pointer" htmlFor="theme-switch">
                                        深色模式
                                    </Label>
                                    <p className="text-xs text-muted-foreground">
                                        目前：{theme === 'dark' ? '深色' : '淺色'}
                                    </p>
                                </div>
                            </div>
                            {/* Switch 元件：checked 對應 dark 模式 */}
                            <Switch
                                id="theme-switch"
                                checked={theme === 'dark'}
                                onCheckedChange={toggleTheme}
                            />
                        </div>

                        {/* 配色風格選擇 */}
                        <div className="flex items-center justify-between px-4 py-3">
                            <div className="flex items-center gap-3">
                                <div className="p-1.5 bg-pink-500/10 text-pink-500 rounded-md">
                                    <Palette size={16} />
                                </div>
                                <div>
                                    <Label className="text-sm font-medium">配色風格</Label>
                                    <p className="text-xs text-muted-foreground">選擇應用程式的色彩主題</p>
                                </div>
                            </div>
                            <Select value={activeThemeId} onValueChange={setThemeId}>
                                <SelectTrigger className="h-8 w-[160px] text-xs">
                                    <SelectValue placeholder="選擇主題" />
                                </SelectTrigger>
                                <SelectContent>
                                    {availableThemes.map(t => (
                                        <SelectItem key={t.id} value={t.id} className="text-xs">
                                            {t.name}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    </CardContent>
                </Card>
            </section>

            {/* ==================== 關於 ==================== */}
            <section className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">關於</h3>

                <Card>
                    <CardContent className="p-0 divide-y divide-border">
                        {/* 更新記錄 */}
                        <Button
                            variant="ghost"
                            onClick={() => setIsChangelogOpen(true)}
                            className="w-full h-auto px-4 py-3 justify-start rounded-none"
                        >
                            <div className="flex items-center gap-3">
                                <div className="p-1.5 bg-emerald-500/10 text-emerald-500 rounded-md">
                                    <FileText size={16} />
                                </div>
                                <div className="text-left">
                                    <p className="text-sm font-medium text-foreground">更新記錄</p>
                                    <p className="text-xs text-muted-foreground">查看版本變更內容</p>
                                </div>
                            </div>
                        </Button>

                        {/* 關於本軟體 */}
                        <Button
                            variant="ghost"
                            onClick={() => setIsAboutOpen(true)}
                            className="w-full h-auto px-4 py-3 justify-start rounded-none"
                        >
                            <div className="flex items-center gap-3">
                                <div className="p-1.5 bg-violet-500/10 text-violet-500 rounded-md">
                                    <Info size={16} />
                                </div>
                                <div className="text-left">
                                    <p className="text-sm font-medium text-foreground">關於 BOMIX</p>
                                    <p className="text-xs text-muted-foreground">版本資訊與授權</p>
                                </div>
                            </div>
                        </Button>
                    </CardContent>
                </Card>
            </section>

            {/* 對話框 */}
            <AboutDialog isOpen={isAboutOpen} onClose={() => setIsAboutOpen(false)} />
            <ChangelogDialog isOpen={isChangelogOpen} onClose={() => setIsChangelogOpen(false)} />
        </div>
    )
}

export default SettingsPage
