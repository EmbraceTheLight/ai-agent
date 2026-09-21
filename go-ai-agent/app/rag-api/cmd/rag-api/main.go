package main

import (
	"flag"
	"log/slog"
	"os"

	"go-ai-agent/app/rag-api/internal/conf"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger, cleanupLogger, err := newLogger()
	if err != nil {
		panic(err)
	}
	defer cleanupLogger()
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource("KRATOS"),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

// newLogger 底层 log 替换为 zap
func newLogger() (*slog.Logger, func(), error) {
	zapConfig := zap.NewProductionConfig()
	zapConfig.OutputPaths = []string{"stdout", "log/log.txt"}
	zapConfig.ErrorOutputPaths = []string{"stderr", "log/error.txt"}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, nil, err
	}

	handler := zapslog.NewHandler(
		zapLogger.Core(),
		zapslog.WithCaller(true),
	)

	logger := log.NewLogger(handler).With(
		slog.String("service.id", id),
		slog.String("service.name", Name),
		slog.String("service.version", Version),
	)

	log.SetDefault(logger)
	slog.SetDefault(logger)

	cleanup := func() {
		_ = zapLogger.Sync()
	}

	return logger, cleanup, nil
}
