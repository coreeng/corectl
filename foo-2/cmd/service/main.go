// main package
package main

import (
	"context"
	"errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreeng/idp-reference-app-go/cmd/handler"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

func main() {
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("failed to create prometheus exporter: %v", err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	router, err := handler.Router()
	if err != nil {
		log.Fatalf("Failed to create router: %+v", err)
	}

	httpServer := &http.Server{
		Addr: ":8080",
		Handler: otelhttp.NewHandler(
			router,
			"gin",
			otelhttp.WithMeterProvider(provider),
		),
	}

	internalServer := &http.Server{
		Addr:    ":8081",
		Handler: handler.InternalRouter(),
	}

	go trapSigTermSignal(httpServer)
	go trapSigTermSignal(internalServer)

	go start(internalServer)
	start(httpServer)
}

func trapSigTermSignal(httpServer *http.Server) {
	log.Info("Watching for SIGTERM signals")
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM)

	sig := <-signalChannel
	log.Info("Received signal: ", sig)

	log.Info("Shutting down gracefully..")

	time.Sleep(time.Second * 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
}

func start(server *http.Server) {
	log.Println("application started")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("Unexpected error %v", err)
	} else {
		log.Println("Application shut down gracefully")
	}
}