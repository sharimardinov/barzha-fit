package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"barzhafit/internal/service"
)

//go:embed assets/*
var assets embed.FS

type Server struct {
	addr     string
	botToken string
	tz       string

	plan      *service.PlanService
	workout   *service.WorkoutService
	targets   *service.TargetsService
	nutrition *service.NutritionService
	steps     *service.StepsService
	profile   *service.ProfileService
	planView  *service.PlanViewService
	statsView *service.StatsViewService
}

type Deps struct {
	Addr      string
	BotToken  string
	TZ        string
	Plan      *service.PlanService
	Workout   *service.WorkoutService
	Targets   *service.TargetsService
	Nutrition *service.NutritionService
	Steps     *service.StepsService
	Profile   *service.ProfileService
	PlanView  *service.PlanViewService
	StatsView *service.StatsViewService
}

func NewServer(d Deps) *Server {
	return &Server{
		addr:      d.Addr,
		botToken:  d.BotToken,
		tz:        d.TZ,
		plan:      d.Plan,
		workout:   d.Workout,
		targets:   d.Targets,
		nutrition: d.Nutrition,
		steps:     d.Steps,
		profile:   d.Profile,
		planView:  d.PlanView,
		statsView: d.StatsView,
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()

	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return fmt.Errorf("miniapp assets: %w", err)
	}
	fileServer := http.FileServer(http.FS(sub))
	serveMiniapp := func(prefix string) {
		mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
		})
		mux.Handle(prefix+"/", http.StripPrefix(prefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" || strings.HasSuffix(path, ".html") {
				w.Header().Set("Cache-Control", "no-store")
			} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
		})))
	}
	serveMiniapp("/miniapp")
	serveMiniapp("/miniapp-v3")

	s.registerAPI(mux)

	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("miniapp listening on %s", s.addr)
	return server.ListenAndServe()
}
