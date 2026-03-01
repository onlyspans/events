package grpc

import (
	"errors"
	"log/slog"

	"github.com/onlyspans/events/internal/apperr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toGRPCError(logger *slog.Logger, err error) error {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		logger.Error("unexpected error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}

	switch appErr.Type {
	case apperr.TypeValidation:
		return status.Error(codes.InvalidArgument, appErr.Error())
	case apperr.TypeNotFound:
		return status.Error(codes.NotFound, appErr.Error())
	case apperr.TypeConflict:
		return status.Error(codes.AlreadyExists, appErr.Error())
	case apperr.TypeInternal:
		logger.Error("internal error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	default:
		logger.Error("unknown error type", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
