package delivery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/tentens-tech/shared-lock/internal/application"
	"github.com/tentens-tech/shared-lock/internal/config"
	httpserver "github.com/tentens-tech/shared-lock/internal/delivery/http"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/cache"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
)

func NewServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		RunE:  sharedLockProcess,
	}
}

func sharedLockProcess(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	errGroup, errGroupCtx := errgroup.WithContext(ctx)

	var runChan = make(chan os.Signal, 1)
	signal.Notify(runChan, os.Interrupt)

	configuration := config.NewConfig()
	if configuration.Debug {
		log.SetLevel(log.DebugLevel)
	}

	var leaseCache *cache.Cache
	if configuration.Cache.Enabled {
		log.Info("Cache is enabled")
		leaseCache = cache.New(configuration.Cache.Size)
	} else {
		log.Info("Cache is disabled")
	}

	errGroup.Go(func() error {
		app := application.New(errGroupCtx, configuration, leaseCache)
		server := httpserver.New(app)

		log.Printf("Server is starting on %s\n", configuration.Server.Port)
		serverErrChan := make(chan error, 1)
		go func() {
			if err := server.Start(&configuration.Server); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Errorf("Server encountered an error: %v", err)
				serverErrChan <- err
				return
			}
			serverErrChan <- nil
		}()

		select {
		case err := <-serverErrChan:
			if err != nil {
				return err
			}
		case interrupt := <-runChan:
			ctxWithTimeout, cancel := context.WithTimeout(
				errGroupCtx,
				configuration.Server.Timeout.Shutdown,
			)
			defer cancel()

			log.Printf("Server is shutting down due to %+v\n", interrupt)
			if err := server.Server.Shutdown(ctxWithTimeout); err != nil {
				log.Errorf("Server was unable to gracefully shutdown due to err: %+v", err)
				return err
			}
		}

		return nil
	})

	return errGroup.Wait()
}
