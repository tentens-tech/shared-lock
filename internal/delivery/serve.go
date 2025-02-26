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

func sharedLockProcess(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	eg := errgroup.Group{}
	// Set up a channel to listen to for interrupt signals
	var runChan = make(chan os.Signal, 1)
	// Handle ctrl+c/ctrl+x interrupt
	signal.Notify(runChan, os.Interrupt)

	configuration := config.NewConfig()

	eg.Go(func() error {
		// Define server options
		server := &http.Server{
			Addr:         ":" + configuration.Server.Port,
			Handler:      application.NewRouter(ctx, configuration),
			ReadTimeout:  configuration.Server.Timeout.Read * time.Second,
			WriteTimeout: configuration.Server.Timeout.Write * time.Second,
			IdleTimeout:  configuration.Server.Timeout.Idle * time.Second,
		}

		// Alert the user that the server is starting
		log.Printf("Server is starting on %s\n", server.Addr)

		// Run the server on a new goroutine
		go func() {
			if err := server.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					// Normal interrupt operation, ignore
				} else {
					log.Fatalf("Server failed to start due to err: %v", err)
				}
			}
		}()

		// Block on this channel listeninf for those previously defined syscalls assign
		// to variable so we can let the user know why the server is shutting down
		interrupt := <-runChan

		// Set up a context to allow for graceful server shutdowns in the event
		// of an OS interrupt (defers the cancel just in case)
		ctxWithTimeout, cancel := context.WithTimeout(
			ctx,
			configuration.Server.Timeout.Shutdown,
		)
		defer cancel()

		// If we get one of the pre-prescribed syscalls, gracefully terminate the server
		// while alerting the user
		log.Printf("Server is shutting down due to %+v\n", interrupt)
		if err := server.Shutdown(ctxWithTimeout); err != nil {
			log.Fatalf("Server was unable to gracefully shutdown due to err: %+v", err)
		}

		return nil
	})

	return eg.Wait()
}
