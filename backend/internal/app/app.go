package app

import (
	"log/slog"
	"net/http"
	"time"

	"selfemi/internal/modules/decision"
	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/modules/pricing"
	"selfemi/internal/platform/httpserver"
)

type Dependencies struct {
	DB        httpserver.Pinger
	GoalsRepo goals.Repository
	Logger    *slog.Logger
	Now       func() time.Time
}

func NewHandler(deps Dependencies) http.Handler {
	goalService := goals.NewService(deps.GoalsRepo, deps.Now)
	decisionService := decision.NewService(
		goalLookup{goals: goalService},
		decision.StubEngine{},
		deps.Now,
		finance.StubStateSource{},
		pricing.NewDecisionInputs(pricing.StubPriceData{}, pricing.StubForecast{}, deps.Now),
	)
	return httpserver.NewRouter(deps.DB,
		goals.NewHandler(goalService, deps.Logger),
		decision.NewHandler(decisionService, deps.Logger),
	)
}
