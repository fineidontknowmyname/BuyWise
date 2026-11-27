package decision

import (
	"context"
	"errors"
	"time"
	"uuid"
)

type State string

const (
	Buy          State = "BUY"
	Wait         State = "WAIT"
	SaveMore     State = "SAVE_MORE"
	Reassess     State = "REASSESS"
	RulesVersion       = "purchase-v1"
)

var ErrGoalNotFound = errors.New("goal not found")
var ErrHistoryUnavailable = errors.New("decision history unavailable")

type GoalRef struct {
	ID          uuid.UUID
	Name        string
	Status      string
	ProductID   *uuid.UUID
	TargetPaise int64
}

type AccountEvidence struct {
	ID                 string `json:"id"`
	Purpose            string `json:"purpose"`
	BalancePaise       int64  `json:"balancePaise"`
	GoalReservedPaise  int64  `json:"goalReservedPaise"`
	OtherReservedPaise int64  `json:"otherReservedPaise"`
}

type FinancialEvidence struct {
	ReviewRequired       bool              `json:"reviewRequired,omitempty"`
	ProfileRevision      int               `json:"profileRevision"`
	DebtStatus           string            `json:"debtStatus"`
	MonthlyMarginPaise   int64             `json:"monthlyMarginPaise"`
	EmergencyTargetPaise int64             `json:"emergencyTargetPaise"`
	OperatingBufferPaise int64             `json:"operatingBufferPaise"`
	Accounts             []AccountEvidence `json:"accounts"`
}

type OfferEvidence struct {
	OfferID         string     `json:"offerId"`
	Retailer        string     `json:"retailer"`
	Seller          string     `json:"seller"`
	ObservationID   string     `json:"observationId"`
	ObservedOn      string     `json:"observedOn"`
	ObservedAt      *time.Time `json:"observedAt"`
	FetchedAt       *time.Time `json:"fetchedAt"`
	Source          string     `json:"source"`
	SourceLabel     string     `json:"sourceLabel"`
	Availability    string     `json:"availability"`
	ItemPaise       *int64     `json:"itemPaise"`
	TaxPaise        *int64     `json:"taxPaise"`
	DeliveryPaise   *int64     `json:"deliveryPaise"`
	DiscountPaise   *int64     `json:"discountPaise"`
	HistoryIDs      []string   `json:"historyIds"`
	HistoryLowPaise int64      `json:"historyLowPaise"`
	HistorySumPaise int64      `json:"historySumPaise"`
}

type PriceEvidence struct {
	ProductID    string          `json:"productId"`
	Variant      string          `json:"variant"`
	IdealPaise   *int64          `json:"idealPaise"`
	GoodPaise    *int64          `json:"goodPaise"`
	MaximumPaise *int64          `json:"maximumPaise"`
	Offers       []OfferEvidence `json:"offers"`
}

type Evidence struct {
	Finances FinancialEvidence `json:"finances"`
	Prices   PriceEvidence     `json:"prices"`
}

type Axis struct {
	State       string   `json:"state"`
	ReasonCodes []string `json:"reasonCodes"`
}

type Reason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Decision struct {
	ID                    string         `json:"id,omitempty"`
	GoalID                string         `json:"goalId"`
	GoalName              string         `json:"goalName"`
	GoalStatus            string         `json:"goalStatus"`
	GoalTargetPaise       int64          `json:"goalTargetPaise"`
	State                 State          `json:"state"`
	RulesVersion          string         `json:"rulesVersion"`
	DecidedAt             time.Time      `json:"decidedAt"`
	Trigger               string         `json:"trigger"`
	Finance               Axis           `json:"finance"`
	Price                 Axis           `json:"price"`
	Evidence              Evidence       `json:"evidence"`
	SelectedOffer         *OfferEvidence `json:"selectedOffer"`
	CheckoutPaise         *int64         `json:"checkoutPaise"`
	BackedGoalPaise       int64          `json:"backedGoalPaise"`
	EmergencyBalancePaise int64          `json:"emergencyBalancePaise"`
	AfterPurchasePaise    *int64         `json:"afterPurchasePaise"`
	ReasonCodes           []string       `json:"reasonCodes"`
	Reasons               []Reason       `json:"reasons"`
	Explanation           string         `json:"explanation"`
	NextSteps             []string       `json:"nextSteps"`
	Assumptions           []string       `json:"assumptions"`
}

type GoalLookup interface {
	Goal(context.Context, uuid.UUID) (GoalRef, error)
}

type InputProvider interface {
	Inputs(context.Context, GoalRef, time.Time) (Evidence, error)
}
