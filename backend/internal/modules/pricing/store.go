package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"selfemi/internal/platform/database"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateProduct(ctx context.Context, p Product) error {
	specsJSON, err := json.Marshal(p.Specifications)
	if err != nil || len(p.Specifications) == 0 {
		specsJSON = []byte("{}")
	}
	warningsJSON, err := json.Marshal(p.Warnings)
	if err != nil || len(p.Warnings) == 0 {
		warningsJSON = []byte("[]")
	}
	if p.Currency == "" {
		p.Currency = "INR"
	}
	if p.Availability == "" {
		p.Availability = "in_stock"
	}
	if p.Source == "" {
		p.Source = "manual"
	}
	if p.VerificationStatus == "" {
		p.VerificationStatus = "unverified"
	}

	_, err = database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO pricing.products (
			id, name, brand, model, variant, category, description,
			current_price_paise, original_price_paise, target_price_paise,
			discount_percent, currency, availability, specifications,
			url, image_url, retailer, notes, last_retrieved_at,
			source, verification_status, warnings, created_at, updated_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13, $14::jsonb,
			$15, $16, $17, $18, $19,
			$20, $21, $22::jsonb, $23, $24
		)`,
		p.ID.String(), p.Name, p.Brand, p.Model, p.Variant, p.Category, p.Description,
		p.CurrentPricePaise, p.OriginalPricePaise, p.TargetPricePaise,
		p.DiscountPercent, p.Currency, p.Availability, specsJSON,
		p.URL, p.ImageURL, p.Retailer, p.Notes, p.LastRetrievedAt,
		p.Source, p.VerificationStatus, warningsJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (s *Store) GetProduct(ctx context.Context, id uuid.UUID) (Product, error) {
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT
			id::text, name, brand, model, variant, category, description,
			current_price_paise, original_price_paise, target_price_paise,
			discount_percent, currency, availability, specifications::text,
			url, image_url, retailer, notes, last_retrieved_at,
			source, verification_status, warnings::text, created_at, updated_at
		FROM pricing.products WHERE id = $1::uuid`,
		id.String(),
	)
	var p Product
	var idStr string
	var specsText, warningsText string
	err := row.Scan(
		&idStr, &p.Name, &p.Brand, &p.Model, &p.Variant, &p.Category, &p.Description,
		&p.CurrentPricePaise, &p.OriginalPricePaise, &p.TargetPricePaise,
		&p.DiscountPercent, &p.Currency, &p.Availability, &specsText,
		&p.URL, &p.ImageURL, &p.Retailer, &p.Notes, &p.LastRetrievedAt,
		&p.Source, &p.VerificationStatus, &warningsText, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product: %w", err)
	}
	p.ID, _ = uuid.Parse(idStr)
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	if p.LastRetrievedAt != nil {
		t := p.LastRetrievedAt.UTC()
		p.LastRetrievedAt = &t
	}
	_ = json.Unmarshal([]byte(specsText), &p.Specifications)
	if p.Specifications == nil {
		p.Specifications = make(map[string]string)
	}
	_ = json.Unmarshal([]byte(warningsText), &p.Warnings)
	if p.Warnings == nil {
		p.Warnings = make([]string, 0)
	}
	return p, nil
}

func (s *Store) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT
			id::text, name, brand, model, variant, category, description,
			current_price_paise, original_price_paise, target_price_paise,
			discount_percent, currency, availability, specifications::text,
			url, image_url, retailer, notes, last_retrieved_at,
			source, verification_status, warnings::text, created_at, updated_at
		FROM pricing.products ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var p Product
		var idStr string
		var specsText, warningsText string
		err := rows.Scan(
			&idStr, &p.Name, &p.Brand, &p.Model, &p.Variant, &p.Category, &p.Description,
			&p.CurrentPricePaise, &p.OriginalPricePaise, &p.TargetPricePaise,
			&p.DiscountPercent, &p.Currency, &p.Availability, &specsText,
			&p.URL, &p.ImageURL, &p.Retailer, &p.Notes, &p.LastRetrievedAt,
			&p.Source, &p.VerificationStatus, &warningsText, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		p.ID, _ = uuid.Parse(idStr)
		p.CreatedAt = p.CreatedAt.UTC()
		p.UpdatedAt = p.UpdatedAt.UTC()
		if p.LastRetrievedAt != nil {
			t := p.LastRetrievedAt.UTC()
			p.LastRetrievedAt = &t
		}
		_ = json.Unmarshal([]byte(specsText), &p.Specifications)
		if p.Specifications == nil {
			p.Specifications = make(map[string]string)
		}
		_ = json.Unmarshal([]byte(warningsText), &p.Warnings)
		if p.Warnings == nil {
			p.Warnings = make([]string, 0)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list products rows: %w", err)
	}
	return products, nil
}

func (s *Store) UpdateProduct(ctx context.Context, p Product) error {
	specsJSON, err := json.Marshal(p.Specifications)
	if err != nil || len(p.Specifications) == 0 {
		specsJSON = []byte("{}")
	}
	warningsJSON, err := json.Marshal(p.Warnings)
	if err != nil || len(p.Warnings) == 0 {
		warningsJSON = []byte("[]")
	}
	if p.Currency == "" {
		p.Currency = "INR"
	}
	if p.Availability == "" {
		p.Availability = "in_stock"
	}

	cmd, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`UPDATE pricing.products SET
			name = $2, brand = $3, model = $4, variant = $5, category = $6,
			description = $7, current_price_paise = $8, original_price_paise = $9,
			target_price_paise = $10, discount_percent = $11, currency = $12,
			availability = $13, specifications = $14::jsonb, url = $15,
			image_url = $16, retailer = $17, notes = $18, last_retrieved_at = $19,
			source = $20, verification_status = $21, warnings = $22::jsonb,
			updated_at = $23
		WHERE id = $1::uuid`,
		p.ID.String(), p.Name, p.Brand, p.Model, p.Variant, p.Category,
		p.Description, p.CurrentPricePaise, p.OriginalPricePaise,
		p.TargetPricePaise, p.DiscountPercent, p.Currency,
		p.Availability, specsJSON, p.URL,
		p.ImageURL, p.Retailer, p.Notes, p.LastRetrievedAt,
		p.Source, p.VerificationStatus, warningsJSON,
		p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (s *Store) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	cmd, err := database.Scoped(ctx, s.pool).Exec(ctx, `DELETE FROM pricing.products WHERE id = $1::uuid`, id.String())
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (s *Store) AddPriceObservation(ctx context.Context, obs PriceObservation) error {
	var batchID *string
	if obs.ImportBatchID != nil {
		str := obs.ImportBatchID.String()
		batchID = &str
	}
	var goalID *string
	if obs.GoalID != nil {
		str := obs.GoalID.String()
		goalID = &str
	}
	if obs.Currency == "" {
		obs.Currency = "INR"
	}
	if obs.PriceType == "" {
		obs.PriceType = "regular"
	}

	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO pricing.price_observations (
			id, product_id, goal_id, price_paise, currency, price_type,
			observed_on, retailer, source, provenance, source_ref,
			notes, import_batch_id, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
			$7::date, $8, $9, $10, $11,
			$12, $13::uuid, $14
		) ON CONFLICT (product_id, observed_on) DO UPDATE SET
			price_paise = EXCLUDED.price_paise,
			currency = EXCLUDED.currency,
			price_type = EXCLUDED.price_type,
			retailer = EXCLUDED.retailer,
			source = EXCLUDED.source,
			provenance = EXCLUDED.provenance,
			source_ref = EXCLUDED.source_ref,
			notes = EXCLUDED.notes`,
		obs.ID.String(), obs.ProductID.String(), goalID, obs.PricePaise, obs.Currency, obs.PriceType,
		obs.ObservedOn, obs.Retailer, obs.Source, obs.Provenance, obs.SourceRef,
		obs.Notes, batchID, obs.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add price observation: %w", err)
	}

	// Update current_price_paise if this observation is the latest
	_, _ = database.Scoped(ctx, s.pool).Exec(ctx,
		`UPDATE pricing.products p SET
			current_price_paise = (
				SELECT price_paise FROM pricing.price_observations
				WHERE product_id = p.id ORDER BY observed_on DESC, id DESC LIMIT 1
			),
			updated_at = NOW()
		WHERE p.id = $1::uuid`,
		obs.ProductID.String(),
	)
	return nil
}

func (s *Store) ListPriceObservations(ctx context.Context, productID uuid.UUID) ([]PriceObservation, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT
			id::text, product_id::text, goal_id::text, price_paise,
			COALESCE(currency, 'INR'), COALESCE(price_type, 'regular'),
			observed_on::text, COALESCE(retailer, ''), source, provenance,
			COALESCE(source_ref, ''), COALESCE(notes, ''),
			import_batch_id::text, created_at
		FROM pricing.price_observations
		WHERE product_id = $1::uuid
		ORDER BY observed_on ASC, id ASC`,
		productID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list price observations: %w", err)
	}
	defer rows.Close()

	return scanObservations(rows)
}

func (s *Store) ListPriceObservationsForGoal(ctx context.Context, goalID uuid.UUID) ([]PriceObservation, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT
			id::text, product_id::text, goal_id::text, price_paise,
			COALESCE(currency, 'INR'), COALESCE(price_type, 'regular'),
			observed_on::text, COALESCE(retailer, ''), source, provenance,
			COALESCE(source_ref, ''), COALESCE(notes, ''),
			import_batch_id::text, created_at
		FROM pricing.price_observations
		WHERE goal_id = $1::uuid
		ORDER BY observed_on ASC, id ASC`,
		goalID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list price observations for goal: %w", err)
	}
	defer rows.Close()

	return scanObservations(rows)
}

func scanObservations(rows pgx.Rows) ([]PriceObservation, error) {
	observations := make([]PriceObservation, 0)
	for rows.Next() {
		var obs PriceObservation
		var idStr, prodStr string
		var goalStr, batchStr *string
		err := rows.Scan(
			&idStr, &prodStr, &goalStr, &obs.PricePaise,
			&obs.Currency, &obs.PriceType,
			&obs.ObservedOn, &obs.Retailer, &obs.Source, &obs.Provenance,
			&obs.SourceRef, &obs.Notes,
			&batchStr, &obs.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan price observation: %w", err)
		}
		obs.ID, _ = uuid.Parse(idStr)
		obs.ProductID, _ = uuid.Parse(prodStr)
		if goalStr != nil {
			gID, _ := uuid.Parse(*goalStr)
			obs.GoalID = &gID
		}
		if batchStr != nil {
			bID, _ := uuid.Parse(*batchStr)
			obs.ImportBatchID = &bID
		}
		obs.CreatedAt = obs.CreatedAt.UTC()
		observations = append(observations, obs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("price observations rows: %w", err)
	}
	return observations, nil
}

func (s *Store) ImportPriceBatch(ctx context.Context, batch ImportBatch, observations []PriceObservation) (ImportBatch, error) {
	tx, err := database.Scoped(ctx, s.pool).Begin(ctx)
	if err != nil {
		return batch, fmt.Errorf("begin import tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO pricing.price_import_batches (
			id, product_id, filename, total_rows, imported_rows, duplicate_rows, error_rows, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8
		)`,
		batch.ID.String(), batch.ProductID.String(), batch.Filename,
		batch.TotalRows, batch.ImportedRows, batch.DuplicateRows, batch.ErrorRows, batch.CreatedAt,
	)
	if err != nil {
		return batch, fmt.Errorf("insert batch record: %w", err)
	}

	imported := 0
	duplicates := 0
	batchIDStr := batch.ID.String()

	for _, obs := range observations {
		var goalID *string
		if obs.GoalID != nil {
			str := obs.GoalID.String()
			goalID = &str
		}
		curr := obs.Currency
		if curr == "" {
			curr = "INR"
		}
		pType := obs.PriceType
		if pType == "" {
			pType = "regular"
		}

		tag, err := tx.Exec(ctx,
			`INSERT INTO pricing.price_observations (
				id, product_id, goal_id, price_paise, currency, price_type,
				observed_on, retailer, source, provenance, source_ref,
				notes, import_batch_id, created_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
				$7::date, $8, $9, $10, $11,
				$12, $13::uuid, $14
			) ON CONFLICT (product_id, observed_on) DO NOTHING`,
			obs.ID.String(), batch.ProductID.String(), goalID, obs.PricePaise, curr, pType,
			obs.ObservedOn, obs.Retailer, obs.Source, obs.Provenance, obs.SourceRef,
			obs.Notes, batchIDStr, obs.CreatedAt,
		)
		if err != nil {
			return batch, fmt.Errorf("insert batch observation: %w", err)
		}
		if tag.RowsAffected() > 0 {
			imported++
		} else {
			duplicates++
		}
	}

	batch.ImportedRows = imported
	batch.DuplicateRows = duplicates

	_, err = tx.Exec(ctx,
		`UPDATE pricing.price_import_batches SET
			imported_rows = $2, duplicate_rows = $3
		WHERE id = $1::uuid`,
		batch.ID.String(), batch.ImportedRows, batch.DuplicateRows,
	)
	if err != nil {
		return batch, fmt.Errorf("update batch counts: %w", err)
	}

	// Update current price
	_, _ = tx.Exec(ctx,
		`UPDATE pricing.products p SET
			current_price_paise = COALESCE((
				SELECT price_paise FROM pricing.price_observations
				WHERE product_id = p.id ORDER BY observed_on DESC, id DESC LIMIT 1
			), p.current_price_paise),
			updated_at = NOW()
		WHERE p.id = $1::uuid`,
		batch.ProductID.String(),
	)

	if err := tx.Commit(ctx); err != nil {
		return batch, fmt.Errorf("commit import tx: %w", err)
	}
	return batch, nil
}

func (s *Store) GetProductStats(ctx context.Context, productID uuid.UUID) (ProductStats, error) {
	prod, err := s.GetProduct(ctx, productID)
	if err != nil {
		return ProductStats{}, err
	}

	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT
			COALESCE(MIN(price_paise), 0),
			COALESCE(MAX(price_paise), 0),
			COALESCE(AVG(price_paise)::bigint, 0),
			COUNT(*),
			COALESCE(MIN(observed_on)::text, ''),
			COALESCE(MAX(observed_on)::text, ''),
			COALESCE((
				SELECT price_paise FROM pricing.price_observations
				WHERE product_id = $1::uuid
				ORDER BY observed_on DESC, id DESC LIMIT 1
			), 0)
		FROM pricing.price_observations
		WHERE product_id = $1::uuid`,
		productID.String(),
	)

	stats := ProductStats{
		ProductID:         productID,
		CurrentPricePaise: prod.CurrentPricePaise,
	}

	err = row.Scan(
		&stats.LowestPricePaise,
		&stats.HighestPricePaise,
		&stats.AveragePricePaise,
		&stats.ObservationCount,
		&stats.FirstObservedOn,
		&stats.LastObservedOn,
		&stats.LatestPricePaise,
	)
	if err != nil {
		return stats, fmt.Errorf("get product stats: %w", err)
	}
	return stats, nil
}

func (s *Store) CreateDocumentUpload(ctx context.Context, doc DocumentUpload) error {
	parsedJSON, _ := json.Marshal(doc.ParsedData)
	if len(parsedJSON) == 0 {
		parsedJSON = []byte("{}")
	}
	summaryJSON, _ := json.Marshal(doc.Summary)
	if len(summaryJSON) == 0 {
		summaryJSON = []byte("{}")
	}
	var prodID *string
	if doc.ProductID != nil {
		str := doc.ProductID.String()
		prodID = &str
	}

	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO pricing.document_uploads (
			id, goal_id, product_id, filename, file_type, file_size,
			purpose, status, parsed_data, summary, created_at, confirmed_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
			$7, $8, $9::jsonb, $10::jsonb, $11, $12
		)`,
		doc.ID.String(), doc.GoalID.String(), prodID, doc.Filename, doc.FileType, doc.FileSize,
		doc.Purpose, doc.Status, parsedJSON, summaryJSON, doc.CreatedAt, doc.ConfirmedAt,
	)
	if err != nil {
		return fmt.Errorf("create document upload: %w", err)
	}
	return nil
}

func (s *Store) GetDocumentUpload(ctx context.Context, id uuid.UUID) (DocumentUpload, error) {
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT
			id::text, goal_id::text, product_id::text, filename, file_type,
			file_size, purpose, status, parsed_data::text, summary::text,
			created_at, confirmed_at
		FROM pricing.document_uploads WHERE id = $1::uuid`,
		id.String(),
	)
	var doc DocumentUpload
	var idStr, goalStr string
	var prodStr *string
	var parsedText, summaryText string
	err := row.Scan(
		&idStr, &goalStr, &prodStr, &doc.Filename, &doc.FileType,
		&doc.FileSize, &doc.Purpose, &doc.Status, &parsedText, &summaryText,
		&doc.CreatedAt, &doc.ConfirmedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentUpload{}, ErrDocumentNotFound
	}
	if err != nil {
		return DocumentUpload{}, fmt.Errorf("get document upload: %w", err)
	}
	doc.ID, _ = uuid.Parse(idStr)
	doc.GoalID, _ = uuid.Parse(goalStr)
	if prodStr != nil {
		pID, _ := uuid.Parse(*prodStr)
		doc.ProductID = &pID
	}
	doc.CreatedAt = doc.CreatedAt.UTC()
	if doc.ConfirmedAt != nil {
		t := doc.ConfirmedAt.UTC()
		doc.ConfirmedAt = &t
	}
	_ = json.Unmarshal([]byte(parsedText), &doc.ParsedData)
	_ = json.Unmarshal([]byte(summaryText), &doc.Summary)
	return doc, nil
}

func (s *Store) ListDocumentUploadsForGoal(ctx context.Context, goalID uuid.UUID) ([]DocumentUpload, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT
			id::text, goal_id::text, product_id::text, filename, file_type,
			file_size, purpose, status, parsed_data::text, summary::text,
			created_at, confirmed_at
		FROM pricing.document_uploads
		WHERE goal_id = $1::uuid
		ORDER BY created_at DESC`,
		goalID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list document uploads: %w", err)
	}
	defer rows.Close()

	uploads := make([]DocumentUpload, 0)
	for rows.Next() {
		var doc DocumentUpload
		var idStr, goalStr string
		var prodStr *string
		var parsedText, summaryText string
		err := rows.Scan(
			&idStr, &goalStr, &prodStr, &doc.Filename, &doc.FileType,
			&doc.FileSize, &doc.Purpose, &doc.Status, &parsedText, &summaryText,
			&doc.CreatedAt, &doc.ConfirmedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan document upload: %w", err)
		}
		doc.ID, _ = uuid.Parse(idStr)
		doc.GoalID, _ = uuid.Parse(goalStr)
		if prodStr != nil {
			pID, _ := uuid.Parse(*prodStr)
			doc.ProductID = &pID
		}
		doc.CreatedAt = doc.CreatedAt.UTC()
		if doc.ConfirmedAt != nil {
			t := doc.ConfirmedAt.UTC()
			doc.ConfirmedAt = &t
		}
		_ = json.Unmarshal([]byte(parsedText), &doc.ParsedData)
		_ = json.Unmarshal([]byte(summaryText), &doc.Summary)
		uploads = append(uploads, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document uploads rows: %w", err)
	}
	return uploads, nil
}

func (s *Store) UpdateDocumentUpload(ctx context.Context, doc DocumentUpload) error {
	parsedJSON, _ := json.Marshal(doc.ParsedData)
	if len(parsedJSON) == 0 {
		parsedJSON = []byte("{}")
	}
	summaryJSON, _ := json.Marshal(doc.Summary)
	if len(summaryJSON) == 0 {
		summaryJSON = []byte("{}")
	}
	var prodID *string
	if doc.ProductID != nil {
		str := doc.ProductID.String()
		prodID = &str
	}

	cmd, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`UPDATE pricing.document_uploads SET
			product_id = $2::uuid, status = $3, parsed_data = $4::jsonb,
			summary = $5::jsonb, confirmed_at = $6
		WHERE id = $1::uuid`,
		doc.ID.String(), prodID, doc.Status, parsedJSON, summaryJSON, doc.ConfirmedAt,
	)
	if err != nil {
		return fmt.Errorf("update document upload: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}
