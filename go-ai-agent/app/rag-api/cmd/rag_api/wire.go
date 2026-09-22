//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/app/rag-api/internal/data"
	"go-ai-agent/app/rag-api/internal/server"
	"go-ai-agent/app/rag-api/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
