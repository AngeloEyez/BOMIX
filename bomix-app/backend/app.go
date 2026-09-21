package backend

import (
	"context"
	"sync"

	"bomix-app/backend/config"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/version"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gorm.io/gorm"
)

// App 代表 BOMIX Wails 核心應用程式橋接結構體
// 負責管理 Wails 生命週期、底層服務依賴（DB、Logger、Config、TaskMgr）以及事件派發
type App struct {
	app             *application.App
	logger          *logger.Logger
	cfg             *config.Config
	taskMgr         *task.Manager
	db              *gorm.DB
	mu              sync.RWMutex
	confirmChans    map[string]chan bool
	confirmMu       sync.Mutex
	aiCancelFunc    context.CancelFunc
	currentAITaskID string
	aiMu            sync.Mutex
}

// NewApp 建立並初始化一個新的 App 實例
//
// 參數：
//   - wailsApp: Wails v3 應用程式實例指標
//   - logger: 結構化日誌記錄器
//   - cfg: 應用程式全域設定
//
// 回傳：
//   - *App: 初始化完成的 App 實例
func NewApp(wailsApp *application.App, logger *logger.Logger, cfg *config.Config) *App {
	app := &App{
		app:          wailsApp,
		logger:       logger,
		cfg:          cfg,
		confirmChans: make(map[string]chan bool),
	}

	// Set logger event callback
	if logger != nil {
		logger.SetEventCallback(app.EmitEvent)
	}

	// Create task manager with app as the event emitter
	app.taskMgr = task.NewManager(logger, app)

	return app
}

// EmitEvent 透過 Wails 執行期環境向前端發送自訂事件
//
// 參數：
//   - event: 事件名稱（如 "task:progress", "time" 等）
//   - data: 攜帶之資料物件
func (a *App) EmitEvent(event string, data interface{}) {
	if a.app != nil {
		a.app.Event.Emit(event, data)
	}
}

// GetContext 取得應用程式 Context（Wails v3 中已由內部機制管理，此處固定回傳 nil）
//
// 回傳：
//   - context.Context: 固定回傳 nil
func (a *App) GetContext() context.Context {
	return nil // Context not used in v3
}

// Quit 安全結束並關閉應用程式視窗與進程
func (a *App) Quit() {
	if a.app != nil {
		a.app.Quit()
	}
}

// GetVersion 取得目前應用程式版本號字串
//
// 回傳：
//   - string: 版本號 (例如 "1.0.0" 或 "dev")
func (a *App) GetVersion() string {
	return version.GetVersion()
}

// GetAppInfo 取得目前應用程式詳細版本與建置元資料
//
// 回傳：
//   - version.Info: 包含版本號、Git Commit 與建置時間的結構體
func (a *App) GetAppInfo() version.Info {
	return version.Get()
}

