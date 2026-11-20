package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"
)

const captureKeyPrefix = "expense-candidate-"

var (
	errInvalidNotification = errors.New("invalid structured expense notification")
	errArchivedCategory    = errors.New("expense category is archived")
	errCandidateNotFound   = errors.New("expense suggestion not found")
	errCandidateReviewed   = errors.New("expense suggestion already reviewed")
	errPossibleDuplicate   = errors.New("possible duplicate expense")
	androidPackage         = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

type expenseFields struct {
	AccountID   string `json:"accountId"`
	CategoryID  string `json:"categoryId"`
	AmountPaise int64  `json:"amountPaise"`
	OccurredOn  string `json:"occurredOn"`
	Note        string `json:"note"`
}

func (i expenseFields) transaction(today time.Time, key string) (NewTransaction, error) {
	account, err := uuid.Parse(i.AccountID)
	if err != nil {
		return NewTransaction{}, ErrAccountNotFound
	}
	category, err := uuid.Parse(i.CategoryID)
	if err != nil {
		return NewTransaction{}, ErrCategoryNotFound
	}
	date, err := time.Parse(DateLayout, i.OccurredOn)
	if err != nil || date.Year() < 1 || date.After(today) {
		return NewTransaction{}, ErrInvalidDate
	}
	if i.AmountPaise < 1 || i.AmountPaise > MaxAmountPaise {
		return NewTransaction{}, ErrInvalidAmount
	}
	if strings.ContainsRune(i.Note, 0) || utf8.RuneCountInString(i.Note) > MaxNoteLength {
		return NewTransaction{}, ErrInvalidNote
	}
	return NewTransaction{AccountID: account, CategoryID: category, Kind: Expense, AmountPaise: i.AmountPaise,
		OccurredOn: i.OccurredOn, Note: strings.TrimSpace(i.Note), IdempotencyKey: key}, nil
}

func (s *Service) captureExpense(ctx context.Context, in NewTransaction) (Transaction, bool, error) {
	if store, ok := s.repo.(*Store); ok && validKey(in.IdempotencyKey) {
		if err := store.lockCapture(ctx); err != nil {
			return Transaction{}, false, err
		}
		prior, err := store.transactionBy(ctx, "idempotency_key", in.IdempotencyKey)
		if err != nil {
			return Transaction{}, false, err
		}
		if prior != nil {
			if !matchesExpense(*prior, in) {
				return Transaction{}, false, ErrKeyConflict
			}
			return *prior, false, nil
		}
	}
	category, err := s.repo.GetCategory(ctx, in.CategoryID)
	if err != nil {
		return Transaction{}, false, err
	}
	if category.Archived {
		return Transaction{}, false, errArchivedCategory
	}
	return s.CreateTransaction(ctx, in)
}

func matchesExpense(tx Transaction, in NewTransaction) bool {
	return tx.Kind == Expense && tx.AccountID == in.AccountID && tx.CategoryID == in.CategoryID &&
		tx.AmountPaise == in.AmountPaise && tx.OccurredOn.Format(DateLayout) == in.OccurredOn && tx.Note == in.Note
}

type notificationInput struct {
	SourceApp   string `json:"sourceApp"`
	Reference   string `json:"reference"`
	Merchant    string `json:"merchant"`
	AmountPaise int64  `json:"amountPaise"`
	Currency    string `json:"currency"`
	OccurredAt  string `json:"occurredAt"`
}

type captureCandidate struct {
	ID                string              `json:"id"`
	SourceApp         string              `json:"sourceApp"`
	Merchant          string              `json:"merchant,omitempty"`
	AmountPaise       int64               `json:"amountPaise"`
	OccurredOn        string              `json:"occurredOn"`
	Status            string              `json:"status"`
	TransactionID     *string             `json:"transactionId,omitempty"`
	PossibleDuplicate bool                `json:"possibleDuplicate"`
	Category          *categorySuggestion `json:"categorySuggestion,omitempty"`
}

func digest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func (i notificationInput) candidate(now time.Time) (captureCandidate, string, *string, error) {
	app := strings.TrimSpace(i.SourceApp)
	merchant := strings.Join(strings.Fields(i.Merchant), " ")
	reference := strings.ToUpper(strings.TrimSpace(i.Reference))
	at, err := time.Parse(time.RFC3339Nano, i.OccurredAt)
	if err != nil || at.Year() < 1 || at.After(now) || i.Currency != "INR" || i.AmountPaise < 1 ||
		i.AmountPaise > MaxAmountPaise || len(app) > 160 || !androidPackage.MatchString(app) ||
		utf8.RuneCountInString(merchant) > 120 || utf8.RuneCountInString(reference) > 120 ||
		strings.ContainsFunc(i.Merchant+i.Reference, unicode.IsControl) {
		return captureCandidate{}, "", nil, errInvalidNotification
	}
	var refHash *string
	if reference != "" {
		hash := digest(reference)
		refHash = &hash
	}
	fingerprint := digest(app, strings.ToLower(merchant), strconv.FormatInt(i.AmountPaise, 10), at.UTC().Format(time.RFC3339))
	return captureCandidate{SourceApp: app, Merchant: merchant, AmountPaise: i.AmountPaise,
		OccurredOn: at.In(businessZone).Format(DateLayout)}, fingerprint, refHash, nil
}

type categorySuggestion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
}

func suggestExpenseCategory(merchant string, categories []Category) *categorySuggestion {
	name := strings.ToLower(merchant)
	if name == "" {
		return nil
	}
	var inferred string
	switch name {
	case "zomato", "swiggy":
		inferred = "dining out"
	case "blinkit", "zepto", "bigbasket", "swiggy instamart":
		inferred = "groceries"
	case "uber", "ola cabs", "rapido":
		inferred = "transport"
	}
	var fallback *categorySuggestion
	for _, category := range categories {
		if category.Kind != Expense || category.Archived {
			continue
		}
		categoryName := strings.ToLower(category.Name)
		if name == categoryName {
			return &categorySuggestion{category.ID.String(), category.Name, "medium"}
		}
		if inferred == categoryName {
			fallback = &categorySuggestion{category.ID.String(), category.Name, "low"}
		}
	}
	return fallback
}
