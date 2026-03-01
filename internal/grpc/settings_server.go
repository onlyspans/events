package grpc

import (
	"context"
	"log/slog"

	eventsv1 "github.com/onlyspans/events/gen/go/events/v1"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/ports"
)

type SettingsServer struct {
	eventsv1.UnimplementedSettingsServiceServer
	settingsService ports.SettingsService
	logger          *slog.Logger
}

func NewSettingsServer(settingsService ports.SettingsService, logger *slog.Logger) *SettingsServer {
	return &SettingsServer{settingsService: settingsService, logger: logger}
}

func (s *SettingsServer) GetSettings(ctx context.Context, _ *eventsv1.GetSettingsRequest) (*eventsv1.GetSettingsResponse, error) {
	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return nil, toGRPCError(s.logger, err)
	}
	return &eventsv1.GetSettingsResponse{
		Settings: dtoToProtoSettings(settings),
	}, nil
}

func (s *SettingsServer) UpdateSettings(ctx context.Context, req *eventsv1.UpdateSettingsRequest) (*eventsv1.UpdateSettingsResponse, error) {
	updated, err := s.settingsService.UpdateSettings(ctx, &dto.SettingsDTO{
		RetentionPeriodDays: int(req.Settings.RetentionPeriodDays),
		MaxExportSize:       int(req.Settings.MaxExportSize),
	})
	if err != nil {
		return nil, toGRPCError(s.logger, err)
	}
	return &eventsv1.UpdateSettingsResponse{
		Settings: dtoToProtoSettings(updated),
	}, nil
}

func dtoToProtoSettings(s *dto.SettingsDTO) *eventsv1.Settings {
	return &eventsv1.Settings{
		RetentionPeriodDays: int32(s.RetentionPeriodDays),
		MaxExportSize:       int32(s.MaxExportSize),
	}
}
