package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"

	"github.com/lib/pq"
)

var ErrProductNotFound = errors.New("product not found")
var ErrCategoryNameTaken = errors.New("category name already exists")
var ErrProductStockLevelsRequired = errors.New("at least one stock location is required")

type StockRepo struct{ db *sql.DB }

func NewStockRepo(db *sql.DB) *StockRepo { return &StockRepo{db: db} }

type ProductStockLevelInput struct {
	LocationID        string
	OnHandQuantity    int
	FreeToUseQuantity int
}

type ProductUpsertInput struct {
	Name        string
	Cost        float64
	CategoryID  string
	Description string
	StockLevels []ProductStockLevelInput
}

type storedProductStockLevel struct {
	LocationID        string `json:"location_id"`
	OnHandQuantity    int    `json:"on_hand_quantity"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
}

func (r *StockRepo) ListCategories() ([]models.ProductCategory, error) {
	const q = `
		SELECT id, name, COALESCE(description, '')
		FROM stocks.categories
		ORDER BY name ASC`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]models.ProductCategory, 0)
	for rows.Next() {
		var category models.ProductCategory
		if err := rows.Scan(&category.ID, &category.Name, &category.Description); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *StockRepo) CreateCategory(name, description string) (*models.ProductCategory, error) {
	const q = `
		INSERT INTO stocks.categories (name, description)
		VALUES ($1, $2)
		RETURNING id, name, COALESCE(description, '')`

	var category models.ProductCategory
	err := r.db.QueryRow(q, name, nullableText(description)).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "categories_name") {
			return nil, ErrCategoryNameTaken
		}
		return nil, err
	}

	return &category, nil
}

func (r *StockRepo) ListLocations() ([]models.ProductLocation, error) {
	const q = `
		SELECT
			l.id,
			l.name,
			l.short_code,
			COALESCE(array_agg(w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names
		FROM locations.locations l
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		GROUP BY l.id, l.name, l.short_code
		ORDER BY l.name ASC`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]models.ProductLocation, 0)
	for rows.Next() {
		var location models.ProductLocation
		var warehouses pq.StringArray
		if err := rows.Scan(&location.ID, &location.Name, &location.ShortCode, &warehouses); err != nil {
			return nil, err
		}
		location.WarehouseNames = append([]string(nil), warehouses...)
		locations = append(locations, location)
	}
	return locations, rows.Err()
}

func (r *StockRepo) ListProducts(search string, limit int) ([]models.Product, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}

	search = strings.TrimSpace(search)
	locationsByID, err := r.loadLocationsByID()
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.cost,
			p.category_id,
			c.name AS category_name,
			COALESCE(p.stock_levels, '[]'::jsonb),
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		WHERE (
			$1 = ''
			OR p.sku ILIKE '%' || $1 || '%'
			OR p.name ILIKE '%' || $1 || '%'
			OR c.name ILIKE '%' || $1 || '%'
			OR COALESCE(p.description, '') ILIKE '%' || $1 || '%'
			OR EXISTS (
				SELECT 1
				FROM jsonb_array_elements(COALESCE(p.stock_levels, '[]'::jsonb)) AS stock_level(value)
				JOIN locations.locations lx ON lx.id::text = stock_level.value->>'location_id'
				WHERE lx.name ILIKE '%' || $1 || '%'
				   OR lx.short_code ILIKE '%' || $1 || '%'
			)
		)
		ORDER BY p.updated_at DESC
		LIMIT $2`

	rows, err := r.db.Query(q, search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		var (
			product        models.Product
			rawStockLevels []byte
		)

		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Cost,
			&product.CategoryID,
			&product.CategoryName,
			&rawStockLevels,
			&product.Description,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := populateProductStockFields(&product, rawStockLevels, locationsByID); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

func (r *StockRepo) CreateProduct(input ProductUpsertInput) (*models.Product, error) {
	levels := normalizeStockLevels(input.StockLevels)
	if len(levels) == 0 {
		return nil, ErrProductStockLevelsRequired
	}

	serializedLevels, err := marshalStockLevels(levels)
	if err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO stocks.products
			(name, cost, category_id, stock_levels, description)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		RETURNING id`

	var productID string
	if err := r.db.QueryRow(
		q,
		input.Name,
		input.Cost,
		input.CategoryID,
		serializedLevels,
		nullableText(input.Description),
	).Scan(&productID); err != nil {
		return nil, err
	}

	return r.GetProductByID(productID)
}

func (r *StockRepo) UpdateProduct(productID string, input ProductUpsertInput) (*models.Product, error) {
	levels := normalizeStockLevels(input.StockLevels)
	if len(levels) == 0 {
		return nil, ErrProductStockLevelsRequired
	}

	serializedLevels, err := marshalStockLevels(levels)
	if err != nil {
		return nil, err
	}

	const q = `
		UPDATE stocks.products
		SET
			name = $2,
			cost = $3,
			category_id = $4,
			stock_levels = $5::jsonb,
			description = $6,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(
		q,
		productID,
		input.Name,
		input.Cost,
		input.CategoryID,
		serializedLevels,
		nullableText(input.Description),
	)
	if err != nil {
		return nil, err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ErrProductNotFound
	}

	return r.GetProductByID(productID)
}

func (r *StockRepo) DeleteProduct(productID string) error {
	const q = `DELETE FROM stocks.products WHERE id = $1`
	result, err := r.db.Exec(q, productID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *StockRepo) GetProductByID(productID string) (*models.Product, error) {
	locationsByID, err := r.loadLocationsByID()
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.cost,
			p.category_id,
			c.name AS category_name,
			COALESCE(p.stock_levels, '[]'::jsonb),
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		WHERE p.id = $1`

	var (
		product        models.Product
		rawStockLevels []byte
	)

	err = r.db.QueryRow(q, productID).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Cost,
		&product.CategoryID,
		&product.CategoryName,
		&rawStockLevels,
		&product.Description,
		&product.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	if err := populateProductStockFields(&product, rawStockLevels, locationsByID); err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *StockRepo) loadLocationsByID() (map[string]models.ProductLocation, error) {
	locations, err := r.ListLocations()
	if err != nil {
		return nil, err
	}

	locationsByID := make(map[string]models.ProductLocation, len(locations))
	for _, location := range locations {
		locationsByID[location.ID] = location
	}
	return locationsByID, nil
}

func populateProductStockFields(product *models.Product, rawStockLevels []byte, locationsByID map[string]models.ProductLocation) error {
	levels, err := unmarshalProductStockLevels(rawStockLevels)
	if err != nil {
		return err
	}

	product.StockLevels = make([]models.ProductStockLevel, 0, len(levels))
	product.OnHandQuantity = 0
	product.FreeToUseQuantity = 0
	product.LocationID = ""
	product.LocationName = ""
	product.LocationShortCode = ""
	product.WarehouseNames = nil

	for index, level := range levels {
		if metadata, ok := locationsByID[level.LocationID]; ok {
			level.LocationName = metadata.Name
			level.LocationShortCode = metadata.ShortCode
			level.WarehouseNames = append([]string(nil), metadata.WarehouseNames...)
		}

		product.StockLevels = append(product.StockLevels, level)
		product.OnHandQuantity += level.OnHandQuantity
		product.FreeToUseQuantity += level.FreeToUseQuantity

		if index == 0 {
			product.LocationID = level.LocationID
			product.LocationName = level.LocationName
			product.LocationShortCode = level.LocationShortCode
			product.WarehouseNames = append([]string(nil), level.WarehouseNames...)
		}
	}

	return nil
}

func marshalStockLevels(levels []ProductStockLevelInput) ([]byte, error) {
	storedLevels := make([]storedProductStockLevel, 0, len(levels))
	for _, level := range levels {
		storedLevels = append(storedLevels, storedProductStockLevel{
			LocationID:        strings.TrimSpace(level.LocationID),
			OnHandQuantity:    level.OnHandQuantity,
			FreeToUseQuantity: level.FreeToUseQuantity,
		})
	}
	return json.Marshal(storedLevels)
}

func unmarshalProductStockLevels(rawStockLevels []byte) ([]models.ProductStockLevel, error) {
	if len(rawStockLevels) == 0 {
		return []models.ProductStockLevel{}, nil
	}

	storedLevels := make([]storedProductStockLevel, 0)
	if err := json.Unmarshal(rawStockLevels, &storedLevels); err == nil {
		levels := make([]models.ProductStockLevel, 0, len(storedLevels))
		for _, storedLevel := range storedLevels {
			levels = append(levels, models.ProductStockLevel{
				LocationID:        strings.TrimSpace(storedLevel.LocationID),
				OnHandQuantity:    storedLevel.OnHandQuantity,
				FreeToUseQuantity: storedLevel.FreeToUseQuantity,
			})
		}

		return levels, nil
	}

	legacyStructured := make(map[string]storedProductStockLevel)
	if err := json.Unmarshal(rawStockLevels, &legacyStructured); err == nil {
		keys := make([]string, 0, len(legacyStructured))
		for locationID := range legacyStructured {
			keys = append(keys, locationID)
		}
		sort.Strings(keys)

		levels := make([]models.ProductStockLevel, 0, len(keys))
		for _, locationID := range keys {
			storedLevel := legacyStructured[locationID]
			normalizedLocationID := strings.TrimSpace(storedLevel.LocationID)
			if normalizedLocationID == "" {
				normalizedLocationID = strings.TrimSpace(locationID)
			}

			levels = append(levels, models.ProductStockLevel{
				LocationID:        normalizedLocationID,
				OnHandQuantity:    storedLevel.OnHandQuantity,
				FreeToUseQuantity: storedLevel.FreeToUseQuantity,
			})
		}

		return levels, nil
	}

	legacyFreeOnly := make(map[string]int)
	if err := json.Unmarshal(rawStockLevels, &legacyFreeOnly); err == nil {
		keys := make([]string, 0, len(legacyFreeOnly))
		for locationID := range legacyFreeOnly {
			keys = append(keys, locationID)
		}
		sort.Strings(keys)

		levels := make([]models.ProductStockLevel, 0, len(keys))
		for _, locationID := range keys {
			levels = append(levels, models.ProductStockLevel{
				LocationID:        strings.TrimSpace(locationID),
				OnHandQuantity:    0,
				FreeToUseQuantity: legacyFreeOnly[locationID],
			})
		}

		return levels, nil
	}

	// Do not fail the entire request due one malformed stock_levels row.
	return []models.ProductStockLevel{}, nil
}

func normalizeStockLevels(stockLevels []ProductStockLevelInput) []ProductStockLevelInput {
	orderedIDs := make([]string, 0, len(stockLevels))
	levelsByLocation := make(map[string]ProductStockLevelInput)

	for _, level := range stockLevels {
		locationID := strings.TrimSpace(level.LocationID)
		if locationID == "" {
			continue
		}

		if _, exists := levelsByLocation[locationID]; !exists {
			orderedIDs = append(orderedIDs, locationID)
		}

		levelsByLocation[locationID] = ProductStockLevelInput{
			LocationID:        locationID,
			OnHandQuantity:    level.OnHandQuantity,
			FreeToUseQuantity: level.FreeToUseQuantity,
		}
	}

	normalized := make([]ProductStockLevelInput, 0, len(orderedIDs))
	for _, locationID := range orderedIDs {
		normalized = append(normalized, levelsByLocation[locationID])
	}

	return normalized
}
