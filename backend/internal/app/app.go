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
	"selfemi/internal/platform/identity"
)

type Dependencies struct {
	Identity    *identity.Handler
	DB          httpserver.Pinger
	GoalsRepo   goals.Repository
	FinanceRepo finance.Repository
	PricingRepo pricing.Repository
	Logger      *slog.Logger
	Now         func() time.Time
}

func newDomainHandler(deps Dependencies) http.Handler {
	pricingRepo := deps.PricingRepo
	if pricingRepo == nil {
		pricingRepo = pricing.NewStubRepository()
	}

	goalService := goals.NewService(deps.GoalsRepo, deps.Now)
	financeService := finance.NewService(deps.FinanceRepo, deps.Now)
	pricingService := pricing.NewService(pricingRepo, deps.Now)

	decisionService := decision.NewService(
		goalLookup{goals: goalService},
		decision.StubEngine{},
		deps.Now,
		finance.StubStateSource{},
		pricing.NewDecisionInputs(pricingService, pricing.StubForecast{}, deps.Now),
	)
	modules := []httpserver.Module{
		goals.NewHandler(goalService, deps.Logger),
		finance.NewHandler(financeService, deps.Logger),
		pricing.NewHandler(pricingService, deps.Logger),
		decision.NewHandler(decisionService, deps.Logger),
	}
	if goalsStore, ok := deps.GoalsRepo.(*goals.Store); ok {
		if financeStore, ok := deps.FinanceRepo.(*finance.Store); ok {
			modules = append(modules, goals.NewPlanner(goalsStore, planningFinance{financeStore}, deps.Logger, deps.Now))
		}
	}
	return httpserver.NewRouter(deps.DB, modules...)

}

func NewHandler(deps Dependencies) http.Handler {
	if deps.Identity == nil {
		panic("identity handler is required")
	}
	domain := newDomainHandler(deps)
	mux := http.NewServeMux()
	deps.Identity.Register(mux)
	public := httpserver.NewRouter(deps.DB)
	mux.Handle("GET /healthz", public)
	mux.Handle("GET /readyz", public)
	mux.Handle("/", deps.Identity.Protect(domain))
	return mux
}
