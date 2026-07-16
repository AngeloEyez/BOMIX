# Project Guidelines: BOMIX (Go Windows Excel Tool)

## 1. Technical Stack
- **Project Name:** BOMIX
- **Backend/Desktop Core:** Go (Golang)
- **GUI Framework:** Wails v3 (Go + Webview)
- **Frontend Stack:** Vite + Vue 3 (Composition API) + Tailwind CSS v4 + PrimeVue v4
- **Excel Engine:** github.com/xuri/excelize/v2

## 2. Workspace Layout & Nested Directory Structure
BOMIX utilizes a nested mono-repo-like structure. All files MUST be placed strictly according to the workspace directory structure shown below based on their specific purpose. If there is any uncertainty about where a file belongs, you MUST ask the user for clarification before creating or placing it. All development documents, global test scripts, and agent workspaces are placed parallel to the core source application folder (`bomix-app`).

```text
BOMIX/                         # Root workspace (Open Claude Code here)
├── CLAUDE.md                  # Global guidelines (This file)
├── roadmap.md                 # Project roadmap & progress checklist
├── docs/                      # Development documentation, API specs, and diagrams
│   ├── product-spec.md        # Product specifications
│   └── ...
├── scripts/                   # Global automation and environment helper scripts
└── bomix-app/                 # Clean Wails v3 source directory (Initialized here)
    ├── main.go                # Wails entry point
    ├── wails.json             # Wails configuration
    ├── go.mod                 # Go module definitions
    ├── template/              # Folder for Excel templates (for output generation)
    │   ├── bigmatrix.xlsx     # BigMatrix export template
    │   └── matrix.xlsx        # Matrix export template
    ├── backend/               # Pure Go Backend Core
    │   ├── app.go             # Wails main bridge & lifecycle
    │   ├── config/            # Configuration system
    │   │   ├── config.go      # Configuration loading/merging logic
    │   │   └── defaults.go    # All default values
    │   ├── db/                # Database layer
    │   │   ├── connection.go  # SQLite connection management
    │   │   ├── models.go      # GORM Model definitions
    │   │   ├── series.go      # Series data operations
    │   │   ├── project.go     # Project data operations
    │   │   ├── revision.go    # BOM Revision data operations
    │   │   ├── part.go        # Part data operations
    │   │   └── matrix.go      # Matrix data operations
    │   ├── excel/             # Excel processing module
    │   │   ├── detector.go    # BOM format auto-detection
    │   │   ├── reader.go      # Excel import main logic (interfaces)
    │   │   ├── reader_ebom.go     # EBOM format reader
    │   │   ├── reader_bigmatrix.go# BigMatrix format reader
    │   │   ├── reader_matrix.go   # Matrix format reader
    │   │   ├── writer.go      # Excel export main logic (interfaces)
    │   │   ├── writer_bigmatrix.go# BigMatrix format writer
    │   │   ├── writer_matrix.go   # Matrix format writer
    │   │   └── template.go    # go:embed template management
    │   ├── processor/         # Data calculation & precise aggregation logic
    │   │   ├── aggregator.go  # Data aggregation logic (Main/2nd merge)
    │   │   └── filter.go      # View filtering logic
    │   ├── task/              # Async task system
    │   │   ├── manager.go     # Task manager
    │   │   ├── task.go        # Task definition and status
    │   │   └── callback.go    # Callback mechanism
    │   ├── logger/            # Logging system
    │   │   ├── logger.go      # Structured logging (based on slog)
    │   │   └── buffer.go      # Log buffer (for UI reading)
    │   └── types/             # Domain shared models & interfaces
    │       ├── domain.go      # Domain type definitions
    │       ├── errors.go      # Custom error types
    │       └── interfaces.go  # Core interface definitions
    └── frontend/              # Vue 3 Frontend App
        ├── index.html
        ├── package.json
        ├── vite.config.ts
        └── src/
            ├── main.ts
            ├── App.vue
            ├── router/        # Vue Router
            ├── stores/        # Pinia state management
            ├── components/    # Common components
            ├── views/         # Page components
            └── services/      # Wails bindings calling
                └── api.ts
```

### File Placement & Naming Conventions:

* **Go Unit Tests**: MUST be named with the suffix `_test.go` and reside in the same package folder as the implementation (e.g., `bomix-app/backend/excel/reader_test.go`).
* **Frontend Unit Tests**: Put all Vitest component tests in `bomix-app/frontend/tests/`. Name them as `*.spec.ts`.
* **Excel Templates**: Place all reusable template XLSX files inside `bomix-app/template/` for centralized asset management.
* **Strict Folder Placement**: You MUST strictly respect this folder structure. For any unrecognized or ambiguous files, you MUST ask the user for clarification before creating or placing them.


## 3. Frontend Specifics (Vue 3 + PrimeVue v4)

* **State Management:** Strictly use Vue 3 Composition API (ref, reactive, computed, watch) within `<script setup>` tags.
* **Component Usage:** Leverage PrimeVue components for complex UI (e.g., use VirtualScroller or DataTable for large Excel previews to ensure smooth DOM rendering).
* **Styling:** Use Tailwind CSS for utility-first styling and layout. Avoid writing custom CSS/SCSS unless absolutely necessary for PrimeVue overrides.
* **Event Binding:** Listen to Wails runtime events (window.runtime.EventsOn) for background Go task progress (like Excel processing percentage) and update Vue refs reactively.
* **Responsive Layout & Theme Switching:** The frontend must support responsive layouts that adapt dynamically to window resizing, and provide theme switching capabilities (e.g., toggling between light and dark modes utilizing PrimeVue and Tailwind CSS).

## 4. Project Architecture & Extensibility (SOLID)

* **Strict Decoupling:** Keep UI layout, Wails binding (app.go), and pure Go business logic strictly separated. app.go MUST only handle Wails lifecycle hooks and bridging events.
* **Interface-Driven:** Define thin interfaces for data processors and storage (e.g., ExcelReader, ExcelWriter). Define consumer-side interfaces.
* **Dependency Injection:** Pass dependencies via constructors (e.g., NewProcessor(reader ExcelReader)). Do not use global variables or init() functions for business logic setup.

## 5. Memory & Performance Management (Critical)

* **Streaming for Batch Operations:** Use `excelize.NewStreamWriter` for batch writing and the `Rows()` iterator for batch reading to avoid loading entire sheets into memory. For reading or writing specific individual cells (non-batch), direct access methods (e.g., `GetCellValue`, `SetCellValue`) are permitted.
* **Batch Processing:** Process and flush data in chunks (e.g., 1000 rows per batch) rather than accumulating massive slices in memory.
* **Slice Pre-allocation:** Always pre-allocate slices with a known capacity to avoid reallocation overhead (e.g., make([]string, 0, 1000)).
* **Resource Cleanup:** Explicitly defer file.Close() and use explicit variable nil-ing to assist Go's Garbage Collector.

## 6. Concurrency, Lifecycle & Thread Safety

* **Asynchronous Task Management (Wails v3 Manager API):** Implement asynchronous task execution using the Wails v3 Manager API. Do not rely on passing `context.Context` for cancellation. All heavy or long-running tasks must execute asynchronously without blocking the UI main thread.
* **Task State & Progress Tracking:** Properly manage and track the state of each active task (e.g., pending, running, completed, failed). Report task state updates and progress details to the UI log window/console using event bindings.
* **Thread Safety:** No shared mutable state. Multiple goroutines must never read/write to the same slice or map without protection. Prefer passing ownership via channels.
* **sync.Pool:** Use sync.Pool for heavy struct or byte buffer allocations in loops to relieve GC pressure.

## 7. Excel Specifics (Formulas & Data Processing)

* **Formula Handling:** Clearly distinguish between reading raw formula strings (GetCellFormula) and evaluated values. Be aware that using Excelize's CalcCellValue on large sheets can be computationally expensive; apply it selectively or calculate formulas in Go memory using streaming chunks.
* **Safe Aggregation:** Perform aggregations (Sum, Average) in Go's memory during stream reading. For financial or precise decimals, strictly use github.com/shopspring/decimal instead of float64 to avoid IEEE 754 precision issues.
* **Formula Writing:** When writing formulas via StreamWriter, ensure proper cell referencing (SetCellFormula) and apply styling separated from data injection.
* **Template Embedding:** NEVER expect template XLSX files to exist as physical files on the user's disk path during runtime. ALWAYS use `//go:embed` to bundle templates from `bomix-app/template/` into the binary and load them via excelize.OpenReader(bytes.NewReader(embeddedBytes)).

## 8. Strict API Verification Rules

* **Verify Before Coding:** NEVER assume or hallucinate the API methods of excelize/v2, Wails, or PrimeVue.
* **Local Go Lookup:** Before calling an unfamiliar Go library function, you MUST use the Bash tool to run `go doc <package> <method>` to inspect the exact signature and comments.
* **Local Frontend Lookup:** For npm packages, ALWAYS use grep or read tools to inspect the TypeScript declaration files (.d.ts) inside `node_modules/` to verify component props and methods. Do not guess.

## 9. Separated Testing (Testability)

* **Table-Driven Tests:** Write standard Go table-driven tests (t.Run) for all business logic, aggregations, and data formatting functions.
* **Backend Logic Isolation:** Use interface mocks (e.g., mockgen) to mock ExcelReader and ExcelWriter. Test the calculation and filtering algorithms (the Processor) 100% independently of Excel file I/O.
* **UI Layer Testing:** Keep frontend UI pure. Test UI components independently using frontend testing frameworks (like Vitest) by mocking the Wails runtime (window.go bindings).

## 10. Modern Error & Logging

* **Structured Logging:** Use slog for all logging. Include contextual attributes (e.g., slog.String("filename", name)).
* **Error Wrapping:** Use %w when wrapping errors for trace tracking.
* **Sentinel Errors:** Define domain-specific errors (e.g., ErrInvalidFileFormat) so the UI can use errors.Is() to map backend errors to localized, user-friendly dialogues.

## 11. Coding Style & Documentation

* **Standard Function Comments:** Every function/method must include standard structured comments (e.g., JSDoc format for Frontend/JS/TS, Go doc format for Go) describing parameters, return values, and behavior.
* **Key Step Explanations:** Provide clear, human-readable explanations inside code blocks for complex or critical implementation steps.
* **Language Requirement:** All explanations, code comments, and output documentation MUST strictly use Traditional Chinese (繁體中文).

## 12. Build & Compilation Rules
- **Single Binary Target**: The final output for production on Windows MUST be a single, standalone `.exe` file with NO external dependency folder structures.
- **Embedded Assets**: All frontend static assets (Vite build output) and Excel templates inside `bomix-app/template/` must be compiled into the binary using Go's `//go:embed` directive.
- **Wails v3 Build Command**: 
  - Development: `wails3 dev` (runs local hot-reload server)
  - Production Build: `wails3 build -platform windows/amd64` (produces the single executable)