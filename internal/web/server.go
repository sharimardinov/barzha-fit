package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

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

	mux.HandleFunc("/miniapp", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/miniapp/", http.StatusMovedPermanently)
	})

	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return fmt.Errorf("miniapp assets: %w", err)
	}
	mux.Handle("/miniapp/", http.StripPrefix("/miniapp/", http.FileServer(http.FS(sub))))

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
