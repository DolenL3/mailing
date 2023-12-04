package main

import (
	"context"
	"fmt"
	"log"
	"mailing/internal/config"
	httpcontroller "mailing/internal/controllers/http-controller"
	"mailing/internal/mailing"
	"mailing/internal/mailing/api/sender"
	"mailing/internal/mailing/storage/postgres"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	err := run()
	if err != nil {
		log.Printf("run app: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// load .env file
	godotenv.Load()
	// Collecting prerequisites.
	gin.SetMode(gin.ReleaseMode)
	ctx := context.Background()
	client := &http.Client{}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return errors.Wrap(err, "initialising logger")
	}
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	// Creating storage realisation via postgres/pgx.
	dbConfig := config.NewDBConfig()
	pgxConfig, err := pgxpool.ParseConfig(fmt.Sprintf("postgresql://%s:%s@%s/%s", dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.DBName))
	if err != nil {
		return errors.Wrap(err, "parsing pgx config")
	}
	pgxConfig.MaxConnIdleTime = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return errors.Wrap(err, "creating new pgx pool")
	}
	storage := postgres.New(pool, dbConfig)

	// Applying the last version of storage schema.
	err = storage.MigrateUp(ctx)
	if err != nil {
		return errors.Wrap(err, "migrating storage up")
	}

	// Creating sender realisation via sender API.
	senderConfig := config.NewSenderConfig()
	sender := sender.New(client, senderConfig)

	// Creating mailing service from collected dependencies.
	service := mailing.New(storage, sender)

	// Creating controllers.
	router := gin.Default()
	httpHandler := httpcontroller.New(router, service, config.NewHTTPConfig())

	eg, ctx := errgroup.WithContext(ctx)
	// Starting service in errgroup, so the program would exit if error occurred in it.
	eg.Go(func() error {
		err := service.StartFetchingMailings(ctx)
		if err != nil {
			return errors.Wrap(err, "fetching mailings")
		}
		return nil
	})
	// Starting all the controllers (one in this case) in errgroup, so the program would exit if all of them die.
	// Just thought that it is a bad idea, considering that service can still function without controllers,
	// but it doesn't realy matter as it is a test task :)
	eg.Go(func() error {
		wg := &sync.WaitGroup{}
		// Starting http handler.
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := httpHandler.Start()
			if err != nil {
				logger.Error(fmt.Sprintf("http handler died\nError: %v", err))
			}
		}()
		wg.Wait()
		return errors.New("All handlers died")
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}
