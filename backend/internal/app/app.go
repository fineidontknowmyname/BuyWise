package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/modules/decision"
	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/modules/pricing"
	"selfemi/internal/platform/httpserver"
	"selfemi/internal/platform/identity"
)

type Dependencies struct {
	Identity      *identity.Handler
	DB            httpserver.Pinger
	GoalsRepo     goals.Repository
	FinanceRepo   finance.Repository
	PricingRepo   pricing.Repository
	PriceProvider pricing.PriceProvider
	Logger        *slog.Logger
	Now           func() time.Time
}

func newDomainHandler(deps Dependencies) http.Handler {
	pricingRepo := deps.PricingRepo
	if pricingRepo == nil {
		pricingRepo = pricing.NewStubRepository()
	}

	goalService := goals.NewService(deps.GoalsRepo, deps.Now)
	financeService := finance.NewService(deps.FinanceRepo, deps.Now)
	pricingService := pricing.NewService(pricingRepo, deps.Now)

	goalsStore, goalsOK := deps.GoalsRepo.(*goals.Store)
	financeStore, financeOK := deps.FinanceRepo.(*finance.Store)
	pricingStore, pricingOK := pricingRepo.(*pricing.Store)
	pool, poolOK := deps.DB.(*pgxpool.Pool)
	var source decision.InputProvider
	var history *decision.Store
	if goalsOK && financeOK && pricingOK && poolOK {
		source = purchaseInputs{financeStore, goalsStore, pricingStore}
		history = decision.NewStore(pool)
	}
	decisionService := decision.NewService(goalLookup{goals: goalService}, source, history, deps.Now)
	financeHandler := finance.NewHandler(financeService, deps.Logger)
	if goalsOK {
		financeHandler.WithAccountReservations(goalsStore.AccountReservations)
	}
	modules := []httpserver.Module{
		goals.NewHandler(goalService, deps.Logger),
		financeHandler,
		pricing.NewHandler(pricingService, deps.Logger, deps.PriceProvider),
		decision.NewHandler(decisionService, deps.Logger),
	}
	if goalsOK && financeOK {
		modules = append(modules, goals.NewPlanner(goalsStore, planningFinance{financeStore}, deps.Logger, deps.Now))
	}
	if goalsOK && financeOK && poolOK {
		modules = append(modules, purchaseLifecycle{pool, goalsStore, financeStore, deps.Logger, deps.Now})
	}
	if poolOK {
		modules = append(modules, workspace{pool, deps.Logger})
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
