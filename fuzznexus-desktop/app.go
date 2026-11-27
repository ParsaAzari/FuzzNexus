package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"

	"fuzznexus-desktop/internal/blobs"
	"fuzznexus-desktop/internal/bridge"
	"fuzznexus-desktop/internal/dock"
	"fuzznexus-desktop/internal/scan"
	"fuzznexus-desktop/internal/store"
	"fuzznexus-desktop/internal/wordlists"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct manages Wails lifecycle and delegates core actions to the bridge Service.
type App struct {
	ctx       context.Context
	service   *bridge.Service
	logger    *slog.Logger
	quitMu    sync.Mutex
	forceQuit bool
}

// NewApp creates a new App application struct.
func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	svc, err := bridge.NewService("", logger)
	if err != nil {
		logger.Error("Failed to initialize bridge service", "err", err)
	}
	return &App{
		service: svc,
		logger:  logger,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.WindowCenter(ctx)
	dock.GeometricCenter(640, 340)
	dock.HideTrafficLights()

	if a.service != nil {
		a.service.SetContext(ctx)
	}
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		runtime.EventsEmit(ctx, "wails:file-drop", paths)
	})
	dock.Init(func(isDark bool) {
		runtime.EventsEmit(ctx, "macos:theme-changed", isDark)
	})
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	a.quitMu.Lock()
	forced := a.forceQuit
	a.quitMu.Unlock()
	if forced {
		return false
	}

	if a.service != nil && a.service.IsScanActive() {
		runtime.EventsEmit(ctx, "app:confirm-close")
		return true
	}
	return false
}

// ConfirmQuit stops any active scan during shutdown and forces the application to quit.
func (a *App) ConfirmQuit() {
	a.quitMu.Lock()
	a.forceQuit = true
	a.quitMu.Unlock()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) shutdown(ctx context.Context) {
	dock.Cleanup()
	if a.service != nil {
		_ = a.service.Close()
	}
}

// SetAppDockTheme dynamically updates the macOS application dock icon to dark or light mode.
func (a *App) SetAppDockTheme(isDark bool) {
	dock.SetDockTheme(isDark)
}

// FinishSplash transitions the window from launcher size (640x340) to the main application size (1200x760).
func (a *App) FinishSplash() {
	if a.ctx != nil {
		runtime.WindowSetMinSize(a.ctx, 1024, 620)
		runtime.WindowSetSize(a.ctx, 1200, 760)
		runtime.WindowCenter(a.ctx)
		dock.GeometricCenter(1200, 760)
		dock.ShowTrafficLights()
	}
}

// HideLauncherControls ensures the window controls are hidden and the window is centered during launcher.
func (a *App) HideLauncherControls() {
	if a.ctx != nil {
		runtime.WindowCenter(a.ctx)
		dock.GeometricCenter(640, 340)
		dock.HideTrafficLights()
	}
}

// OpenURL opens the given URL in the default system browser.
func (a *App) OpenURL(targetURL string) error {
	if a.ctx != nil && targetURL != "" {
		runtime.BrowserOpenURL(a.ctx, targetURL)
	}
	return nil
}

// StartScan starts a new fuzzing scan session and returns its session ID.
func (a *App) StartScan(req bridge.ScanRequestDTO) (int64, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.StartScan(req)
}

// PauseScan pauses the currently active scan.
func (a *App) PauseScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.PauseScan()
}

// ResumeScan resumes a paused scan.
func (a *App) ResumeScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ResumeScan()
}

// StopScan stops the currently active scan.
func (a *App) StopScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.StopScan()
}

// GetRecentErrors returns recent network errors recorded during the active scan.
func (a *App) GetRecentErrors() []scan.NetworkErrorDTO {
	if a.service == nil {
		return []scan.NetworkErrorDTO{}
	}
	return a.service.GetRecentErrors()
}

// TestConnection probes the target to verify connectivity after a safe-stop event.
func (a *App) TestConnection() (scan.SafeStopTestResultDTO, error) {
	if a.service == nil {
		return scan.SafeStopTestResultDTO{Success: false, Error: "bridge service is not available"}, nil
	}
	return a.service.TestConnection()
}

// ResumeSafeStop resumes a scan that was paused by the safe-stop circuit breaker.
func (a *App) ResumeSafeStop() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ResumeSafeStop()
}

// ResumeAnyway forces a safe-stopped scan to resume regardless of probe status.
func (a *App) ResumeAnyway() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ResumeAnyway()
}

// CancelSafeStop cancels a safe-stopped scan and finalizes its session.
func (a *App) CancelSafeStop() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.CancelSafeStop()
}

// GetResultsSince returns paginated scan results after lastID for live UI streaming.
func (a *App) GetResultsSince(sessionID int64, lastID int64, limit int) (bridge.ResultsPageDTO, error) {
	if a.service == nil {
		return bridge.ResultsPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.GetResultsSince(sessionID, lastID, limit)
}

// GetResultPayload retrieves the stored request/response payload for a result.
func (a *App) GetResultPayload(sessionID int64, resultID int64, maxBytes int) (*blobs.PayloadDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetResultPayload(sessionID, resultID, maxBytes)
}

// AddLiveFilter applies a filter rule dynamically to the running scan.
func (a *App) AddLiveFilter(name, value string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.AddLiveFilter(name, value)
}

// UpdateRate adjusts the requests-per-second throttle of the running scan.
func (a *App) UpdateRate(rps int) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateRate(rps)
}

// GetSessions lists recent scan sessions from persistent storage.
func (a *App) GetSessions(limit int) ([]store.SessionDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetSessions(limit)
}

// GetSession retrieves metadata for a specific scan session by ID.
func (a *App) GetSession(sessionID int64) (*store.SessionDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetSession(sessionID)
}

// ExportResults writes session results to targetPath in the specified format.
func (a *App) ExportResults(sessionID int64, format string, targetPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ExportResults(sessionID, format, targetPath)
}

// GetResultsFiltered returns paginated results matching the given filter group.
func (a *App) GetResultsFiltered(sessionID int64, group store.FilterGroup, limit int, offset int) (bridge.FilteredResultsPageDTO, error) {
	if a.service == nil {
		return bridge.FilteredResultsPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.GetResultsFiltered(sessionID, group, limit, offset)
}

// SaveFilterPreset persists a named filter preset.
func (a *App) SaveFilterPreset(name, filterJSON string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.SaveFilterPreset(name, filterJSON)
}

// GetFilterPresets returns all saved filter presets.
func (a *App) GetFilterPresets() ([]store.FilterPresetDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetFilterPresets()
}

// DeleteFilterPreset removes a saved filter preset by name.
func (a *App) DeleteFilterPreset(name string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DeleteFilterPreset(name)
}

// ListWordlists returns all registered wordlists.
func (a *App) ListWordlists() ([]wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ListWordlists()
}

// CreateWordlist creates a new wordlist with the given name and content.
func (a *App) CreateWordlist(name, initialContent string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CreateWordlist(name, initialContent)
}

// CreateWordlistWithCategory creates a new wordlist assigned to a category.
func (a *App) CreateWordlistWithCategory(name, initialContent, category string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CreateWordlistWithCategory(name, initialContent, category)
}

// GetWordlistCategories returns all available wordlist categories.
func (a *App) GetWordlistCategories() ([]wordlists.CategoryDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetWordlistCategories()
}

// CreateWordlistCategory creates a new wordlist category.
func (a *App) CreateWordlistCategory(id, name, color string) (*wordlists.CategoryDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CreateWordlistCategory(id, name, color)
}

// UpdateWordlistCategory updates the name and color of an existing category.
func (a *App) UpdateWordlistCategory(id, name, color string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateWordlistCategory(id, name, color)
}

// DeleteWordlistCategory deletes a wordlist category by ID.
func (a *App) DeleteWordlistCategory(id string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DeleteWordlistCategory(id)
}

// UpdateWordlist renames an existing wordlist.
func (a *App) UpdateWordlist(id int64, name string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateWordlist(id, name)
}

// UpdateWordlistMetadata updates a wordlist name and category.
func (a *App) UpdateWordlistMetadata(id int64, name string, category string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateWordlistMetadata(id, name, category)
}

// MergeWordlists combines multiple wordlists into a new persistent wordlist.
func (a *App) MergeWordlists(ids []int64, targetName string, category string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.MergeWordlists(ids, targetName, category)
}

// MergeWordlistsToSandbox merges and sanitizes wordlists into a sandboxed file.
func (a *App) MergeWordlistsToSandbox(ids []int64, opts wordlists.SanitizeOptions, isEphemeral bool, targetName string, category string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.MergeWordlistsToSandbox(ids, opts, isEphemeral, targetName, category)
}

// PreviewMergeStats computes deduplication and sanitization stats without writing to disk.
func (a *App) PreviewMergeStats(ids []int64, opts wordlists.SanitizeOptions) (*wordlists.MergeReportDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.PreviewMergeStats(ids, opts)
}

// CleanupSandboxWordlist removes a temporary sandboxed wordlist file.
func (a *App) CleanupSandboxWordlist(path string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.CleanupSandboxWordlist(path)
}

// DeleteWordlist removes a wordlist by ID.
func (a *App) DeleteWordlist(id int64) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DeleteWordlist(id)
}

// DeleteWordlistsBatch removes multiple wordlists by ID and returns the deleted count.
func (a *App) DeleteWordlistsBatch(ids []int64) (int, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.DeleteWordlistsBatch(ids)
}

// ResetBuiltinWordlist restores a built-in wordlist to its default embedded contents.
func (a *App) ResetBuiltinWordlist(id int64) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ResetBuiltinWordlist(id)
}

// RestoreAllBuiltinWordlists restores all built-in wordlists to their defaults.
func (a *App) RestoreAllBuiltinWordlists() ([]*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.RestoreAllBuiltinWordlists()
}

// NormalizeTargetURL parses and normalizes a raw target URL for fuzzing.
func (a *App) NormalizeTargetURL(rawURL string) (*scan.TargetAnalysis, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.NormalizeTargetURL(rawURL)
}

// ReadWordlistPage returns a paginated slice of lines from a wordlist.
func (a *App) ReadWordlistPage(id int64, offset int, limit int) (wordlists.WordlistPageDTO, error) {
	if a.service == nil {
		return wordlists.WordlistPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.ReadWordlistPage(id, offset, limit)
}

// SaveWordlistContent overwrites the contents of a wordlist.
func (a *App) SaveWordlistContent(id int64, content string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.SaveWordlistContent(id, content)
}

// ImportWordlistFromPath copies an external wordlist file into the managed library.
func (a *App) ImportWordlistFromPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ImportWordlistFromPath(sourcePath)
}

// LinkWordlistPath registers an external wordlist file by reference without copying.
func (a *App) LinkWordlistPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.LinkWordlistPath(sourcePath)
}

// RelocateWordlist updates the file path of an existing wordlist.
func (a *App) RelocateWordlist(id int64, newPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.RelocateWordlist(id, newPath)
}

// GetWordlistStats returns line count, size, and metadata stats for a wordlist.
func (a *App) GetWordlistStats(id int64) (*wordlists.WordlistStatsDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetWordlistStats(id)
}

// ExportWordlist copies a managed wordlist to targetPath.
func (a *App) ExportWordlist(id int64, targetPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ExportWordlist(id, targetPath)
}

// VerifyWordlist checks the health and readability of a managed wordlist.
func (a *App) VerifyWordlist(id int64) (*wordlists.WordlistHealthDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.VerifyWordlist(id)
}

// VerifyWordlistPath checks the health and readability of a wordlist at path.
func (a *App) VerifyWordlistPath(path string) (*wordlists.WordlistHealthDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.VerifyWordlistPath(path)
}

// CleanWordlist sanitizes a managed wordlist in place.
func (a *App) CleanWordlist(id int64) (*wordlists.CleanResultDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CleanWordlist(id)
}

// CleanWordlistPath sanitizes a wordlist file at path.
func (a *App) CleanWordlistPath(path string) (*wordlists.CleanResultDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CleanWordlistPath(path)
}

// AnalyzeWordlistRisk scans a wordlist for potentially risky or WAF-triggering payloads.
func (a *App) AnalyzeWordlistRisk(path string) (*wordlists.WordlistRiskReportDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.AnalyzeWordlistRisk(path)
}

// CreateSanitizedSandboxWordlist creates a sanitized copy of sourcePath in the sandbox.
func (a *App) CreateSanitizedSandboxWordlist(sourcePath string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CreateSanitizedSandboxWordlist(sourcePath)
}

// SelectWordlistDialog opens a native file picker to select a wordlist.
func (a *App) SelectWordlistDialog(title string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SelectWordlistDialog(title)
}

// SaveWordlistDialog opens a native save dialog for exporting a wordlist.
func (a *App) SaveWordlistDialog(defaultName string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SaveWordlistDialog(defaultName)
}

// SaveExportDialog opens a native save dialog for exporting scan results.
func (a *App) SaveExportDialog(defaultName, format string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SaveExportDialog(defaultName, format)
}

// SelectAuxiliaryFileDialog opens a native file picker for auxiliary config or TLS files.
func (a *App) SelectAuxiliaryFileDialog(title, filterName, filterPattern string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SelectAuxiliaryFileDialog(title, filterName, filterPattern)
}

// SelectDirectoryDialog opens a native directory picker.
func (a *App) SelectDirectoryDialog(title string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SelectDirectoryDialog(title)
}

// ParseRawHTTPRequest parses a raw HTTP request block into structured scan options.
func (a *App) ParseRawHTTPRequest(rawText, defaultProto string) (*bridge.ParsedRawHTTPRequestDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ParseRawHTTPRequest(rawText, defaultProto)
}

// GetAvailableEncoders returns the list of supported payload encoder names.
func (a *App) GetAvailableEncoders() []string {
	if a.service == nil {
		return []string{}
	}
	return a.service.GetAvailableEncoders()
}

// ImportTOMLConfig loads a TOML config file into a ScanRequestDTO.
func (a *App) ImportTOMLConfig(path string) (*bridge.ScanRequestDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ImportTOMLConfig(path)
}

// SearchFNHASH looks up a historical scan hash in the local history store.
func (a *App) SearchFNHASH(hash string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SearchFNHASH(hash)
}

// GetPendingResumeState returns any interrupted scan state eligible for resumption.
func (a *App) GetPendingResumeState() (*bridge.PendingResumeDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetPendingResumeState()
}

// DiscardPendingResumeState deletes the pending resume state file.
func (a *App) DiscardPendingResumeState() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DiscardPendingResumeState()
}

// ResumePendingScan resumes an interrupted scan from saved state.
func (a *App) ResumePendingScan() (int64, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.ResumePendingScan()
}

// DeleteSessions deletes multiple scan sessions by ID.
func (a *App) DeleteSessions(ids []int64) (int64, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.DeleteSessions(ids)
}

// ClearSessions deletes all stored scan sessions.
func (a *App) ClearSessions() (int64, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.ClearSessions()
}
