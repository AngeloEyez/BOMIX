import { useState } from 'react'
import { Info, FileText, Moon, Sun, Palette } from 'lucide-react'
import useSettingsStore from '../stores/useSettingsStore'
import AboutDialog from '../components/dialogs/AboutDialog'
import ChangelogDialog from '../components/dialogs/ChangelogDialog'

function SettingsPage() {
    const { theme, toggleTheme, activeThemeId, availableThemes, setThemeId } = useSettingsStore()
    const [isAboutOpen, setIsAboutOpen] = useState(false)
    const [isChangelogOpen, setIsChangelogOpen] = useState(false)

    return (
        <div className="max-w-2xl mx-auto space-y-8 animate-fade-in">
            <div>
                <h2 className="text-2xl font-bold text-slate-800 dark:text-white mb-2">設定</h2>
                <p className="text-slate-500 dark:text-slate-400">
                    管理應用程式偏好設定與查看資訊。
                </p>
            </div>

            <div className="space-y-4">
                <h3 className="text-sm font-semibold text-slate-500 uppercase tracking-wider">
                    一般
                </h3>
                
                <div className="bg-white dark:bg-surface-800 rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 overflow-hidden">
                    {/* 主題設定 */}
                    <div className="flex items-center justify-between p-4 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors">
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-lg">
                                {theme === 'light' ? <Sun size={20} /> : <Moon size={20} />}
                            </div>
                            <div>
                                <h4 className="font-medium text-slate-900 dark:text-slate-100">外觀主題</h4>
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    目前設定：{theme === 'light' ? '淺色模式' : '深色模式'}
                                </p>
                            </div>
                        </div>
                        <button 
                            onClick={toggleTheme}
                            className="px-4 py-2 text-sm font-medium text-primary-600 bg-primary-50 hover:bg-primary-100 dark:text-primary-400 dark:bg-primary-900/30 dark:hover:bg-primary-900/50 rounded-lg transition-colors"
                        >
                            切換
                        </button>
                    </div>

                    {/* 主題配色 */}
                    <div className="flex items-center justify-between p-4 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors border-t border-slate-100 dark:border-slate-700">
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-pink-50 dark:bg-pink-900/30 text-pink-600 dark:text-pink-400 rounded-lg">
                                <Palette size={20} />
                            </div>
                            <div>
                                <h4 className="font-medium text-slate-900 dark:text-slate-100">配色風格</h4>
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    選擇應用程式的色彩主題
                                </p>
                            </div>
                        </div>
                        <select
                            value={activeThemeId}
                            onChange={(e) => setThemeId(e.target.value)}
                            className="px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 bg-white dark:bg-surface-700 border border-slate-200 dark:border-slate-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                        >
                            {availableThemes.map(t => (
                                <option key={t.id} value={t.id}>
                                    {t.name}
                                </option>
                            ))}
                        </select>
                    </div>
                </div>
            </div>

            <div className="space-y-4">
                <h3 className="text-sm font-semibold text-slate-500 uppercase tracking-wider">
                    關於
                </h3>

                <div className="bg-white dark:bg-surface-800 rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 overflow-hidden divide-y divide-slate-100 dark:divide-slate-700">
                    {/* 更新記錄 */}
                    <button 
                        onClick={() => setIsChangelogOpen(true)}
                        className="w-full flex items-center justify-between p-4 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors text-left"
                    >
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-emerald-50 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400 rounded-lg">
                                <FileText size={20} />
                            </div>
                            <div>
                                <h4 className="font-medium text-slate-900 dark:text-slate-100">更新記錄</h4>
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    查看版本變更內容
                                </p>
                            </div>
                        </div>
                    </button>

                    {/* 關於本軟體 */}
                    <button 
                        onClick={() => setIsAboutOpen(true)}
                        className="w-full flex items-center justify-between p-4 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors text-left"
                    >
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-purple-50 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 rounded-lg">
                                <Info size={20} />
                            </div>
                            <div>
                                <h4 className="font-medium text-slate-900 dark:text-slate-100">關於 BOMIX</h4>
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    版本資訊與授權
                                </p>
                            </div>
                        </div>
                    </button>
                </div>
            </div>

            <AboutDialog isOpen={isAboutOpen} onClose={() => setIsAboutOpen(false)} />
            <ChangelogDialog isOpen={isChangelogOpen} onClose={() => setIsChangelogOpen(false)} />
        </div>
    )
}

export default SettingsPage
