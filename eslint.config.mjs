import globals from 'globals'
import pluginJs from '@eslint/js'
import pluginReact from 'eslint-plugin-react'
import pluginReactHooks from 'eslint-plugin-react-hooks'

// ========================================
// ESLint 配置（Flat Config 格式）
// ========================================

export default [
    // 全域忽略
    {
        ignores: ['out/**', 'dist/**', 'node_modules/**', 'tests/**', 'agent-workspace/**']
    },

    // JavaScript 基礎規則
    pluginJs.configs.recommended,

    // React 設定
    {
        files: ['src/renderer/**/*.{js,jsx}'],
        plugins: {
            react: pluginReact,
            'react-hooks': pluginReactHooks,
        },
        languageOptions: {
            globals: {
                ...globals.browser,
            },
            parserOptions: {
                ecmaFeatures: { jsx: true },
            },
        },
        settings: {
            react: { version: 'detect' },
        },
        rules: {
            ...pluginReact.configs.recommended.rules,
            ...pluginReactHooks.configs.recommended.rules,
            'react/react-in-jsx-scope': 'off', // React 19 不需要手動 import React
            'react/prop-types': 'off', // 不強制 PropTypes（使用 JSDoc 替代）
            'react-hooks/rules-of-hooks': 'error',
            'react-hooks/exhaustive-deps': 'warn',
        },
    },

    // 主行程 / Preload 設定
    {
        files: ['src/main/**/*.js', 'src/preload/**/*.js'],
        languageOptions: {
            globals: {
                ...globals.node,
                ...globals.browser,
            },
        },
    },

    // 通用規則
    {
        rules: {
            'no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
            'no-console': 'off', // Electron 開發經常需要 console
        },
    },
]
