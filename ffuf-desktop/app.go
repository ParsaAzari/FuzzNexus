package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"ffuf-desktop/internal/blobs"
	"ffuf-desktop/internal/bridge"
	"ffuf-desktop/internal/store"
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
