package app

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

type workspace struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

type workspacePreferences struct {
	DisplayName   string `json:"displayName"`
	PriceUpdates  bool   `json:"priceUpdates"`
	ReducedMotion bool   `json:"reducedMotion"`
}

type activityEvent struct {
	Key        string    `json:"key"`
	Kind       string    `json:"kind"`
	At         time.Time `json:"at"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	Paise      *int64    `json:"paise"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
}

const activityQuery = `WITH events(key,kind,at,title,detail,paise,resource,resource_id) AS (
 SELECT 'tx:'||t.id,'ledger',t.created_at,CASE WHEN t.kind='income' THEN 'Income recorded' ELSE 'Expense recorded' END,
 a.name||' · '||t.occurred_on,t.amount_paise,'account',t.account_id FROM finance.transactions t JOIN finance.accounts a ON a.id=t.account_id
 UNION ALL SELECT 'void:'||t.id,'ledger',t.voided_at,'Transaction voided',a.name,t.amount_paise,'account',t.account_id
 FROM finance.transactions t JOIN finance.accounts a ON a.id=t.account_id WHERE t.voided_at IS NOT NULL
 UNION ALL SELECT 'transfer:'||t.id,'ledger',t.created_at,'Internal transfer',a.name||' → '||b.name,t.amount_paise,'finances',t.id
 FROM finance.transfers t JOIN finance.accounts a ON a.id=t.from_account_id JOIN finance.accounts b ON b.id=t.to_account_id
 UNION ALL SELECT 'funding:'||f.id,'goals',f.created_at,'Contribution earmarked',g.name,f.amount_paise,'plan',g.id
 FROM goals.funding f JOIN goals.goals g ON g.id=f.goal_id
 UNION ALL SELECT 'plan:'||p.goal_id||':'||p.revision,'goals',p.created_at,'Savings plan updated',g.name||' · revision '||p.revision,p.approved_paise,'plan',g.id
 FROM goals.plan_revisions p JOIN goals.goals g ON g.id=p.goal_id
 UNION ALL SELECT 'decision:'||d.id,'decisions',d.decided_at,'Purchase decision saved',g.name||' · '||d.state,NULL,'goal',g.id
 FROM decision.purchase_decisions d JOIN goals.goals g ON g.id=d.goal_id
 UNION ALL SELECT 'offer:'||o.id,'prices',o.recorded_at,'Retailer price recorded',p.name||' · '||r.retailer,o.item_paise,'offers',p.id
 FROM pricing.offer_observations o JOIN pricing.retailer_offers r ON r.id=o.offer_id JOIN pricing.products p ON p.id=r.product_id
 UNION ALL SELECT 'price:'||o.id,'prices',o.created_at,'Product price recorded',p.name||' · '||o.currency,
 CASE WHEN o.currency='INR' THEN o.price_paise END,'product',p.id FROM pricing.price_observations o JOIN pricing.products p ON p.id=o.product_id
 UNION ALL SELECT 'purchase:'||p.goal_id,'purchases',p.created_at,'Purchase completed',g.name,t.amount_paise,'goal',g.id
 FROM goals.purchases p JOIN goals.goals g ON g.id=p.goal_id JOIN finance.transactions t ON t.id=p.transaction_id
 UNION ALL SELECT 'allocation:'||s.goal_id||':'||s.request_key,'purchases',s.created_at,'Surplus allocated',g.name,NULL,'goal',g.id
 FROM goals.surplus_allocations s JOIN goals.goals g ON g.id=s.goal_id
) SELECT key,kind,at,title,detail,paise,resource,resource_id::text FROM events
 WHERE ($1='' OR kind=$1) AND ($2::timestamptz IS NULL OR (at,key)<($2,$3)) ORDER BY at DESC,key DESC LIMIT 21`

func (s workspace) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /activity", s.activity)
	mux.HandleFunc("GET /profile/preferences", s.preferences)
	mux.HandleFunc("PUT /profile/preferences", s.preferences)
}

func (s workspace) activity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind, cursor := q.Get("kind"), q.Get("before")
	switch kind {
	case "", "ledger", "goals", "decisions", "prices", "purchases":
	default:
		httpjson.WriteError(w, 400, "invalid_filter", "Choose an activity group.")
		return
	}
	var before *time.Time
	var key string
	if cursor != "" {
		stamp, value, ok := strings.Cut(cursor, "|")
		at, err := time.Parse(time.RFC3339Nano, stamp)
		if !ok || err != nil || value == "" || len(value) > 120 {
			httpjson.WriteError(w, 400, "invalid_cursor", "Invalid activity cursor.")
			return
		}
		before, key = &at, value
	}
	db := database.Scoped(r.Context(), s.pool)
	rows, err := db.Query(r.Context(), activityQuery, kind, before, key)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()
	events := make([]activityEvent, 0, 21)
	for rows.Next() {
		var event activityEvent
		if err := rows.Scan(&event.Key, &event.Kind, &event.At, &event.Title, &event.Detail, &event.Paise, &event.Resource, &event.ResourceID); err != nil {
			s.fail(w, err)
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, err)
		return
	}
	next := ""
	if len(events) > 20 {
		events = events[:20]
		last := events[19]
		next = last.At.Format(time.RFC3339Nano) + "|" + last.Key
	}
	httpjson.Write(w, 200, map[string]any{"events": events, "next": next})
}

func (s workspace) preferences(w http.ResponseWriter, r *http.Request) {
	db := database.Scoped(r.Context(), s.pool)
	var value workspacePreferences
	if r.Method == http.MethodPut {
		if httpjson.Decode(w, r, &value) != nil || utf8.RuneCountInString(value.DisplayName) > 100 || strings.ContainsRune(value.DisplayName, 0) {
			httpjson.WriteError(w, 400, "invalid_preferences", "Enter a display name of up to 100 characters.")
			return
		}
		value.DisplayName = strings.TrimSpace(value.DisplayName)
		_, err := db.Exec(r.Context(), `INSERT INTO identity.preferences(display_name,price_updates,reduced_motion) VALUES ($1,$2,$3)
		ON CONFLICT (user_id) DO UPDATE SET display_name=EXCLUDED.display_name,price_updates=EXCLUDED.price_updates,reduced_motion=EXCLUDED.reduced_motion`, value.DisplayName, value.PriceUpdates, value.ReducedMotion)
		if err != nil {
			s.fail(w, err)
			return
		}
	} else {
		err := db.QueryRow(r.Context(), `SELECT COALESCE(p.display_name,''),COALESCE(p.price_updates,true),COALESCE(p.reduced_motion,false)
		FROM (SELECT identity.current_user_id() AS id) u LEFT JOIN identity.preferences p ON p.user_id=u.id`).Scan(&value.DisplayName, &value.PriceUpdates, &value.ReducedMotion)
		if err != nil {
			s.fail(w, err)
			return
		}
	}
	httpjson.Write(w, 200, value)
}

func (s workspace) fail(w http.ResponseWriter, err error) {
	s.logger.Error("workspace request failed", "error", err)
	httpjson.WriteError(w, 503, "workspace_unavailable", "BuyWise could not load workspace records.")
}
