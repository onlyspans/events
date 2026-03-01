package grpc

import (
	"log/slog"

	eventsv1 "github.com/onlyspans/events/gen/go/events/v1"
	"github.com/onlyspans/events/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewServer(eventService ports.EventService, settingsService ports.SettingsService, logger *slog.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(logger),
			LoggingInterceptor(logger),
		),
	)
	eventsv1.RegisterEventServiceServer(srv, NewEventServer(eventService, logger))
	eventsv1.RegisterSettingsServiceServer(srv, NewSettingsServer(settingsService, logger))
	reflection.Register(srv)
	return srv
}
