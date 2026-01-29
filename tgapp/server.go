package tgapp

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"barzhafit/backend/service"
)

//go:embed assets/*
var assets embed.FS

//go:embed authweb/*
var authAssets embed.FS

type Server struct {
	addr               string
	botToken           string
	authBotToken       string
	googleClientID     string
	googleClientSecret string
	tz                 string

	plan          *service.PlanService
	workout       *service.WorkoutService
	targets       *service.TargetsService
	nutrition     *service.NutritionService
	steps         *service.StepsService
	profile       *service.ProfileService
	training      *service.TrainingProfileService
	inputs        *service.TrainingInputService
	programs      *service.TrainingProgramService
	injuries      *service.InjuryTypeService
	activity      *service.ActivityAI
	workoutTimer  *service.WorkoutTimerService
	strengthStats *service.WorkoutStatsService
	googleAuth    *service.GoogleAuthService
	sessions      *sessionStore
}

type Deps struct {
	Addr               string
	BotToken           string
	AuthBotToken       string
	GoogleClientID     string
	GoogleClientSecret string
	TZ                 string
	Plan               *service.PlanService
	Workout            *service.WorkoutService
	Targets            *service.TargetsService
	Nutrition          *service.NutritionService
	Steps              *service.StepsService
	Profile            *service.ProfileService
	Training           *service.TrainingProfileService
	Inputs             *service.TrainingInputService
	Programs           *service.TrainingProgramService
	Injuries           *service.InjuryTypeService
	Activity           *service.ActivityAI
	WorkoutTimer       *service.WorkoutTimerService
	StrengthStats      *service.WorkoutStatsService
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
		workout:            d.Workout,
		targets:            d.Targets,
		nutrition:          d.Nutrition,
		steps:              d.Steps,
		profile:            d.Profile,
		training:           d.Training,
		inputs:             d.Inputs,
		programs:           d.Programs,
		injuries:           d.Injuries,
		activity:           d.Activity,
		workoutTimer:       d.WorkoutTimer,
		strengthStats:      d.StrengthStats,
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
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return fmt.Errorf("miniapp assets: %w", err)
	}
	fileServer := http.FileServer(http.FS(sub))
	serveMiniapp := func(prefix string) {
		mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
			target := prefix + "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
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
