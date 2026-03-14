package repository

import (
	"database/sql"
	"errors"
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

	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.cost,
			COALESCE(levels.total_on_hand, p.on_hand_quantity, 0) AS on_hand_quantity,
			COALESCE(levels.total_free_to_use, p.free_to_use_quantity, 0) AS free_to_use_quantity,
			p.category_id,
			c.name AS category_name,
			p.location_id,
			COALESCE(l.name, ''),
			COALESCE(l.short_code, ''),
			COALESCE(array_agg(DISTINCT w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		LEFT JOIN locations.locations l ON l.id = p.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		LEFT JOIN LATERAL (
			SELECT
				SUM(psl.on_hand_quantity)::INT AS total_on_hand,
				SUM(psl.free_to_use_quantity)::INT AS total_free_to_use
			FROM stocks.product_stock_levels psl
			WHERE psl.product_id = p.id
		) levels ON TRUE
		WHERE (
			$1 = ''
			OR p.sku ILIKE '%' || $1 || '%'
			OR p.name ILIKE '%' || $1 || '%'
			OR c.name ILIKE '%' || $1 || '%'
			OR COALESCE(p.description, '') ILIKE '%' || $1 || '%'
			OR EXISTS (
				SELECT 1
				FROM stocks.product_stock_levels pslx
				JOIN locations.locations lx ON lx.id = pslx.location_id
				WHERE pslx.product_id = p.id
				  AND (lx.name ILIKE '%' || $1 || '%' OR lx.short_code ILIKE '%' || $1 || '%')
			)
		)
		GROUP BY
			p.id, p.sku, p.name, p.cost,
			levels.total_on_hand, levels.total_free_to_use,
			p.on_hand_quantity, p.free_to_use_quantity,
			p.category_id, c.name,
			p.location_id, l.name, l.short_code,
			p.description, p.updated_at
		ORDER BY p.updated_at DESC
		LIMIT $2`

	rows, err := r.db.Query(q, search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		var product models.Product
		var warehouses pq.StringArray
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Cost,
			&product.OnHandQuantity,
			&product.FreeToUseQuantity,
			&product.CategoryID,
			&product.CategoryName,
			&product.LocationID,
			&product.LocationName,
			&product.LocationShortCode,
			&warehouses,
			&product.Description,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}
		product.WarehouseNames = append([]string(nil), warehouses...)
		product.StockLevels = make([]models.ProductStockLevel, 0)
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.attachStockLevels(products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *StockRepo) CreateProduct(input ProductUpsertInput) (*models.Product, error) {
	levels := normalizeStockLevels(input.StockLevels)
	if len(levels) == 0 {
		return nil, ErrProductStockLevelsRequired
	}

	primary := levels[0]

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const insertProduct = `
		INSERT INTO stocks.products
			(name, cost, on_hand_quantity, free_to_use_quantity, category_id, location_id, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var productID string
	if err := tx.QueryRow(
		insertProduct,
		input.Name,
		input.Cost,
		primary.OnHandQuantity,
		primary.FreeToUseQuantity,
		input.CategoryID,
		primary.LocationID,
		nullableText(input.Description),
	).Scan(&productID); err != nil {
		return nil, err
	}

	if err := r.replaceProductStockLevelsTx(tx, productID, levels); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetProductByID(productID)
}

func (r *StockRepo) UpdateProduct(productID string, input ProductUpsertInput) (*models.Product, error) {
	levels := normalizeStockLevels(input.StockLevels)
	if len(levels) == 0 {
		return nil, ErrProductStockLevelsRequired
	}

	primary := levels[0]

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const updateProduct = `
		UPDATE stocks.products
		SET
			name = $2,
			cost = $3,
			on_hand_quantity = $4,
			free_to_use_quantity = $5,
			category_id = $6,
			location_id = $7,
			description = $8,
			updated_at = NOW()
		WHERE id = $1`

	result, err := tx.Exec(
		updateProduct,
		productID,
		input.Name,
		input.Cost,
		primary.OnHandQuantity,
		primary.FreeToUseQuantity,
		input.CategoryID,
		primary.LocationID,
		nullableText(input.Description),
	)
	if err != nil {
		return nil, err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ErrProductNotFound
	}

	if err := r.replaceProductStockLevelsTx(tx, productID, levels); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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
	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.cost,
			COALESCE(levels.total_on_hand, p.on_hand_quantity, 0) AS on_hand_quantity,
			COALESCE(levels.total_free_to_use, p.free_to_use_quantity, 0) AS free_to_use_quantity,
			p.category_id,
			c.name AS category_name,
			p.location_id,
			COALESCE(l.name, ''),
			COALESCE(l.short_code, ''),
			COALESCE(array_agg(DISTINCT w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		LEFT JOIN locations.locations l ON l.id = p.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		LEFT JOIN LATERAL (
			SELECT
				SUM(psl.on_hand_quantity)::INT AS total_on_hand,
				SUM(psl.free_to_use_quantity)::INT AS total_free_to_use
			FROM stocks.product_stock_levels psl
			WHERE psl.product_id = p.id
		) levels ON TRUE
		WHERE p.id = $1
		GROUP BY
			p.id, p.sku, p.name, p.cost,
			levels.total_on_hand, levels.total_free_to_use,
			p.on_hand_quantity, p.free_to_use_quantity,
			p.category_id, c.name,
			p.location_id, l.name, l.short_code,
			p.description, p.updated_at`

	var product models.Product
	var warehouses pq.StringArray
	err := r.db.QueryRow(q, productID).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Cost,
		&product.OnHandQuantity,
		&product.FreeToUseQuantity,
		&product.CategoryID,
		&product.CategoryName,
		&product.LocationID,
		&product.LocationName,
		&product.LocationShortCode,
		&warehouses,
		&product.Description,
		&product.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	product.WarehouseNames = append([]string(nil), warehouses...)
	product.StockLevels = make([]models.ProductStockLevel, 0)

	levelsByProduct, err := r.listStockLevelsByProductIDs([]string{productID})
	if err != nil {
		return nil, err
	}
	product.StockLevels = append(product.StockLevels, levelsByProduct[productID]...)

	return &product, nil
}

func (r *StockRepo) replaceProductStockLevelsTx(tx *sql.Tx, productID string, levels []ProductStockLevelInput) error {
	if _, err := tx.Exec(`DELETE FROM stocks.product_stock_levels WHERE product_id = $1`, productID); err != nil {
		return err
	}

	const upsertLevel = `
		INSERT INTO stocks.product_stock_levels (
			product_id,
			location_id,
			on_hand_quantity,
			free_to_use_quantity,
			updated_at
		)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (product_id, location_id)
		DO UPDATE SET
			on_hand_quantity = EXCLUDED.on_hand_quantity,
			free_to_use_quantity = EXCLUDED.free_to_use_quantity,
			updated_at = NOW()`

	for _, level := range levels {
		if _, err := tx.Exec(
			upsertLevel,
			productID,
			level.LocationID,
			level.OnHandQuantity,
			level.FreeToUseQuantity,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *StockRepo) attachStockLevels(products []models.Product) error {
	if len(products) == 0 {
		return nil
	}

	productIDs := make([]string, 0, len(products))
	indexByID := make(map[string]int, len(products))
	for index := range products {
		productIDs = append(productIDs, products[index].ID)
		indexByID[products[index].ID] = index
	}

	levelsByProduct, err := r.listStockLevelsByProductIDs(productIDs)
	if err != nil {
		return err
	}

	for productID, levels := range levelsByProduct {
		if index, ok := indexByID[productID]; ok {
			products[index].StockLevels = append(products[index].StockLevels, levels...)
		}
	}

	return nil
}

func (r *StockRepo) listStockLevelsByProductIDs(productIDs []string) (map[string][]models.ProductStockLevel, error) {
	levelsByProduct := make(map[string][]models.ProductStockLevel)
	if len(productIDs) == 0 {
		return levelsByProduct, nil
	}

	const q = `
		SELECT
			psl.product_id,
			psl.location_id,
			l.name,
			l.short_code,
			COALESCE(array_agg(w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			psl.on_hand_quantity,
			psl.free_to_use_quantity
		FROM stocks.product_stock_levels psl
		JOIN locations.locations l ON l.id = psl.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE psl.product_id = ANY($1::uuid[])
		GROUP BY
			psl.product_id,
			psl.location_id,
			l.name,
			l.short_code,
			psl.on_hand_quantity,
			psl.free_to_use_quantity
		ORDER BY l.name ASC, psl.location_id ASC`

	rows, err := r.db.Query(q, pq.Array(productIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			productID  string
			level      models.ProductStockLevel
			warehouses pq.StringArray
		)
		if err := rows.Scan(
			&productID,
			&level.LocationID,
			&level.LocationName,
			&level.LocationShortCode,
			&warehouses,
			&level.OnHandQuantity,
			&level.FreeToUseQuantity,
		); err != nil {
			return nil, err
		}
		level.WarehouseNames = append([]string(nil), warehouses...)
		levelsByProduct[productID] = append(levelsByProduct[productID], level)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return levelsByProduct, nil
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
