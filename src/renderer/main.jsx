import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// ========================================
// React 應用程式進入點
// 將 App 掛載到 DOM 的 #root 節點
// ========================================

ReactDOM.createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>
)
