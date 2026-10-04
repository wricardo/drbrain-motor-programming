// Command drbrain-motor-programming serves Dr Brain - Motor Programming, the robot puzzle: GraphQL API with
// websocket subscriptions, /llms.txt and the SvelteKit UI.
package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/wricardo/drbrain-motor-programming/api"
	"github.com/wricardo/drbrain-motor-programming/game/config"
	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/game/session"
	"github.com/wricardo/drbrain-motor-programming/transport/websocket"
)

const (
	appName = "drbrain-motor-programming"

	defaultMaxSessions = 1000
	shutdownTimeout    = 10 * time.Second
)

//go:embed llms.txt.tmpl
var llmsTxtSource string

var (
	port         = flag.Int("port", 8000, "HTTP server port")
	host         = flag.String("host", "0.0.0.0", "HTTP server host")
	mapsDir      = flag.String("maps-dir", "maps", "Directory containing map JSON files")
	sessionsDir  = flag.String("sessions-dir", "sessions", "Directory for persisted sessions")
	solutionsDir = flag.String("solutions-dir", "solutions", "Directory containing reference solutions (used by `make validate`; never served)")
	debug        = flag.Bool("debug", false, "Enable debug logging")
	publicURL    = flag.String("public-url", "", "Public base URL rendered into /llms.txt (defaults to the request host)")
)

// envBool reads a boolean env var; unset returns def. "true", "1", "yes" are
// true, anything else is false.
func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "true", "1", "yes":
		return true
	}
	return false
}

// envInt reads a positive integer env var; unset or invalid returns def.
func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("ignoring invalid %s=%q, using %d", key, v, def)
		return def
	}
	return n
}

// envDuration reads a Go duration env var (e.g. "168h"); unset or invalid
// returns def.
func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		log.Printf("ignoring invalid %s=%q, using %s", key, v, def)
		return def
	}
	return d
}

func main() {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("warning: loading .env: %v", err)
		}
	} else {
		log.Println("loaded environment variables from .env")
	}
	flag.Parse()

	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	if err := run(); err != nil {
		log.Fatalf("%s: %v", appName, err)
	}
}

// run wires the application and blocks until SIGINT/SIGTERM.
func run() error {
	maps, err := config.NewManager(*mapsDir)
	if err != nil {
		return fmt.Errorf("load maps: %w", err)
	}
	persistence, err := session.NewFilePersistence(*sessionsDir)
	if err != nil {
		return fmt.Errorf("session persistence: %w", err)
	}
	sessions := session.NewManager(persistence, envInt("MAX_SESSIONS", defaultMaxSessions))
	if err := sessions.LoadAll(); err != nil {
		log.Printf("warning: loading persisted sessions: %v", err)
	}

	hub := websocket.NewHub()
	svc := service.New(maps, sessions, hub, service.Options{
		SessionTTL: envDuration("SESSION_TTL", 7*24*time.Hour),
	})

	introspection := envBool("GRAPHQL_INTROSPECTION", true)
	playground := envBool("GRAPHQL_PLAYGROUND", true)
	log.Printf("GraphQL introspection: %t, playground: %t", introspection, playground)

	apiServer, err := api.NewServer(svc, hub, api.Options{
		Introspection:  introspection,
		Playground:     playground,
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		PublicURL:      *publicURL,
		LLMSTemplate:   llmsTxtSource,
	})
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           apiServer,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s (maps=%s sessions=%s solutions=%s)", addr, *mapsDir, *sessionsDir, *solutionsDir)
		log.Printf("GraphQL: http://%s/graphql  llms.txt: http://%s/llms.txt", addr, addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Printf("received %v, shutting down", sig)
	case err := <-serveErr:
		svc.Shutdown()
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	svc.Shutdown()
	log.Println("server stopped")
	return nil
}
