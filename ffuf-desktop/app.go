package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"ffuf-desktop/internal/blobs"
	"ffuf-desktop/internal/bridge"
	"ffuf-desktop/internal/store"
	"ffuf-desktop/internal/wordlists"
)

// App struct manages Wails lifecycle and delegates core actions to the bridge Service.
type App struct {
	ctx     context.Context
	service *bridge.Service
	logger  *slog.Logger
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
	if a.service != nil {
		a.service.SetContext(ctx)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.service != nil {
		_ = a.service.Close()
	}
}

func (a *App) StartScan(req bridge.ScanRequestDTO) (int64, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.StartScan(req)
}

func (a *App) PauseScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.PauseScan()
}

func (a *App) ResumeScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ResumeScan()
}

func (a *App) StopScan() error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.StopScan()
}

func (a *App) GetResultsSince(sessionID int64, lastID int64, limit int) (bridge.ResultsPageDTO, error) {
	if a.service == nil {
		return bridge.ResultsPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.GetResultsSince(sessionID, lastID, limit)
}

func (a *App) GetResultPayload(sessionID int64, resultID int64, maxBytes int) (*blobs.PayloadDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetResultPayload(sessionID, resultID, maxBytes)
}

func (a *App) AddLiveFilter(name, value string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.AddLiveFilter(name, value)
}

func (a *App) UpdateRate(rps int) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateRate(rps)
}

func (a *App) GetSessions(limit int) ([]store.SessionDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetSessions(limit)
}

func (a *App) GetSession(sessionID int64) (*store.SessionDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetSession(sessionID)
}

func (a *App) ExportResults(sessionID int64, format string, targetPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ExportResults(sessionID, format, targetPath)
}

func (a *App) GetResultsFiltered(sessionID int64, group store.FilterGroup, limit int, offset int) (bridge.FilteredResultsPageDTO, error) {
	if a.service == nil {
		return bridge.FilteredResultsPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.GetResultsFiltered(sessionID, group, limit, offset)
}

func (a *App) SaveFilterPreset(name, filterJSON string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.SaveFilterPreset(name, filterJSON)
}

func (a *App) GetFilterPresets() ([]store.FilterPresetDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetFilterPresets()
}

func (a *App) DeleteFilterPreset(name string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DeleteFilterPreset(name)
}

func (a *App) ListWordlists() ([]wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ListWordlists()
}

func (a *App) CreateWordlist(name, initialContent string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CreateWordlist(name, initialContent)
}

func (a *App) UpdateWordlist(id int64, name string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.UpdateWordlist(id, name)
}

func (a *App) DeleteWordlist(id int64) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.DeleteWordlist(id)
}

func (a *App) DeleteWordlistsBatch(ids []int64) (int, error) {
	if a.service == nil {
		return 0, errors.New("bridge service is not available")
	}
	return a.service.DeleteWordlistsBatch(ids)
}

func (a *App) ReadWordlistPage(id int64, offset int, limit int) (wordlists.WordlistPageDTO, error) {
	if a.service == nil {
		return wordlists.WordlistPageDTO{}, errors.New("bridge service is not available")
	}
	return a.service.ReadWordlistPage(id, offset, limit)
}

func (a *App) SaveWordlistContent(id int64, content string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.SaveWordlistContent(id, content)
}

func (a *App) ImportWordlistFromPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.ImportWordlistFromPath(sourcePath)
}

func (a *App) LinkWordlistPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.LinkWordlistPath(sourcePath)
}

func (a *App) RelocateWordlist(id int64, newPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.RelocateWordlist(id, newPath)
}

func (a *App) GetWordlistStats(id int64) (*wordlists.WordlistStatsDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.GetWordlistStats(id)
}

func (a *App) ExportWordlist(id int64, targetPath string) error {
	if a.service == nil {
		return errors.New("bridge service is not available")
	}
	return a.service.ExportWordlist(id, targetPath)
}

func (a *App) VerifyWordlist(id int64) (*wordlists.WordlistHealthDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.VerifyWordlist(id)
}

func (a *App) CleanWordlist(id int64) (*wordlists.CleanResultDTO, error) {
	if a.service == nil {
		return nil, errors.New("bridge service is not available")
	}
	return a.service.CleanWordlist(id)
}

func (a *App) SelectWordlistDialog(title string) (string, error) {
	if a.service == nil {
		return "", errors.New("bridge service is not available")
	}
	return a.service.SelectWordlistDialog(title)
}
