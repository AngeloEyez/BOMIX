import { resolve } from 'path'
import { defineConfig, externalizeDepsPlugin } from 'electron-vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// ========================================
// Electron-Vite 建置配置
// 定義主行程、Preload、渲染層的 Vite 設定
// ========================================

export default defineConfig({
    // --- 主行程配置 ---
    main: {
        plugins: [externalizeDepsPlugin({ exclude: ['@electron-toolkit/utils'] })],
        build: {
            outDir: 'dist/.intermediate/main',
            rollupOptions: {
                // better-sqlite3 需要作為外部依賴，避免被打包
                external: ['better-sqlite3']
            }
        }
    },

    // --- Preload 腳本配置 ---
    preload: {
        plugins: [externalizeDepsPlugin()],
        build: {
            outDir: 'dist/.intermediate/preload'
        }
    },

    // --- 渲染層配置（React + Tailwind） ---
    renderer: {
        resolve: {
            alias: {
                '@': resolve('src/renderer')
            }
        },
        plugins: [
            react(),
            tailwindcss()
        ],
        build: {
            outDir: 'dist/.intermediate/renderer'
        }
    }
})
