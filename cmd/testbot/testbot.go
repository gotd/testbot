package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func run(ctx context.Context, lg *zap.Logger) (rerr error) {
	appID, err := strconv.Atoi(os.Getenv("APP_ID"))
	if err != nil {
		return errors.Wrapf(err, "APP_ID not set or invalid %q", os.Getenv("APP_ID"))
	}
	appHash := os.Getenv("APP_HASH")
	if appHash == "" {
		return errors.New("no APP_HASH provided")
	}
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return errors.New("no BOT_TOKEN provided")
	}
	// Setting up session storage.
	cacheDir := "/cache"
	sessionDir := filepath.Join(cacheDir, ".td")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return errors.Wrap(err, "mkdir")
	}
	stateDb, err := bolt.Open(filepath.Join(sessionDir, "gaps-state.bbolt"), fs.ModePerm, bolt.DefaultOptions)
	if err != nil {
		return errors.Wrap(err, "state database")
	}
	defer func() {
		if rerr != nil {
			multierr.AppendInto(&rerr, stateDb.Close())
		}
	}()

	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		return nil
	})
	client := telegram.NewClient(appID, appHash, telegram.Options{
		Logger:        lg.Named("client"),
		UpdateHandler: dispatcher,
		SessionStorage: &session.FileStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
	})
	return client.Run(ctx, func(ctx context.Context) error {
		lg.Debug("Client initialized")

		au := client.Auth()
		status, err := au.Status(ctx)
		if err != nil {
			return errors.Wrap(err, "auth status")
		}

		if !status.Authorized {
			if _, err := au.Bot(ctx, token); err != nil {
				return errors.Wrap(err, "login")
			}
			// Refresh auth status.
			if status, err = au.Status(ctx); err != nil {
				return errors.Wrap(err, "auth status")
			}
			lg.Info("Logged in as bot",
				zap.String("name", status.User.Username),
			)
		} else {
			lg.Info("Bot login restored",
				zap.String("name", status.User.Username),
			)
		}
		<-ctx.Done()
		return ctx.Err()
	})
}

func main() {
	cfg := zap.NewProductionConfig()
	if s := os.Getenv("OTEL_LOG_LEVEL"); s != "" {
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(s)); err != nil {
			panic(err)
		}
		cfg.Level.SetLevel(lvl)
	}
	lg, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	if err := run(context.Background(), lg); err != nil {
		lg.Error("Failed", zap.Error(err))
		os.Exit(1)
	}
}
