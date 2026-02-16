package tgapp

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"barzhafit/backend/service"
)

//go:embed assets/*
var _ embed.FS // old miniapp assets, kept for embed but no longer served

//go:embed frontend/dist/*
var reactAssets embed.FS

//go:embed authweb/*
var authAssets embed.FS

type Server struct {
	addr               string
	botToken           string
	authBotToken       string
	googleClientID     string
	googleClientSecret string
	tz                 string

	plan         *service.PlanService
	targets      *service.TargetsService
	nutrition    *service.NutritionService
	steps        *service.StepsService
	profile      *service.ProfileService
	training     *service.TrainingProfileService
	workoutTimer *service.WorkoutTimerService
	googleAuth   *service.GoogleAuthService
	sessions     *sessionStore
}

type Deps struct {
	Addr               string
	BotToken           string
	AuthBotToken       string
	GoogleClientID     string
	GoogleClientSecret string
	TZ                 string
	Plan               *service.PlanService
	Targets            *service.TargetsService
	Nutrition          *service.NutritionService
	Steps              *service.StepsService
	Profile            *service.ProfileService
	Training           *service.TrainingProfileService
	WorkoutTimer       *service.WorkoutTimerService
	GoogleAuth         *service.GoogleAuthService
}

func NewServer(d Deps) *Server {
	secret := d.AuthBotToken
	if secret == "" {
		secret = d.BotToken
	}
	return &Server{
		addr:               d.Addr,
		botToken:           d.BotToken,
		authBotToken:       d.AuthBotToken,
		googleClientID:     d.GoogleClientID,
		googleClientSecret: d.GoogleClientSecret,
		tz:                 d.TZ,
		plan:               d.Plan,
		targets:            d.Targets,
		nutrition:          d.Nutrition,
		steps:              d.Steps,
		profile:            d.Profile,
		training:           d.Training,
		workoutTimer:       d.WorkoutTimer,
		googleAuth:         d.GoogleAuth,
		sessions:           newSessionStore(secret),
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()

	authSub, err := fs.Sub(authAssets, "authweb")
	if err != nil {
		return fmt.Errorf("auth assets: %w", err)
	}
	// Redirect old /miniapp/ to /app/
	mux.HandleFunc("/miniapp", func(w http.ResponseWriter, r *http.Request) {
		target := "/app/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.Handle("/miniapp/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "/app/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}))

	// Serve React build at /app/
	reactSub, err := fs.Sub(reactAssets, "frontend/dist")
	if err != nil {
		return fmt.Errorf("react assets: %w", err)
	}
	reactFS := http.FileServer(http.FS(reactSub))
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		target := "/app/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.Handle("/app/", http.StripPrefix("/app/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		// SPA fallback: serve index.html for non-asset routes
		if path != "" && !strings.Contains(path, ".") {
			path = ""
		}
		if path == "" {
			// Serve index.html directly to avoid FileServer's redirect loop
			data, err := fs.ReadFile(reactSub, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		reactFS.ServeHTTP(w, r)
	})))

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/login" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(authSub, "login.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/login/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
	})

	s.registerAuth(mux)
	s.registerAPI(mux)

	server := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("miniapp listening on %s", s.addr)
	return server.ListenAndServe()
}
