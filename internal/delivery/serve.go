package delivery

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/tentens-tech/shared-lock/internal/application"
	"github.com/tentens-tech/shared-lock/internal/config"
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

	errGroup.Go(func() error {
		server := &http.Server{
			Addr:         ":" + configuration.Server.Port,
			Handler:      application.NewRouter(errGroupCtx, configuration),
			ReadTimeout:  configuration.Server.Timeout.Read * time.Second,
			WriteTimeout: configuration.Server.Timeout.Write * time.Second,
			IdleTimeout:  configuration.Server.Timeout.Idle * time.Second,
		}

		log.Printf("Server is starting on %s\n", server.Addr)
		go func() {
			if err := server.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					// Normal interrupt operation, ignore
				} else {
					log.Fatalf("Server failed to start due to err: %v", err)
				}
			}
		}()

		interrupt := <-runChan

		ctxWithTimeout, cancel := context.WithTimeout(
			errGroupCtx,
			configuration.Server.Timeout.Shutdown,
		)
		defer cancel()

		log.Printf("Server is shutting down due to %+v\n", interrupt)
		if err := server.Shutdown(ctxWithTimeout); err != nil {
			log.Fatalf("Server was unable to gracefully shutdown due to err: %+v", err)
		}

		return nil
	})

	return errGroup.Wait()
}
