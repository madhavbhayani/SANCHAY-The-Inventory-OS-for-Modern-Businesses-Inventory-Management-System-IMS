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

type StockRepo struct{ db *sql.DB }

func NewStockRepo(db *sql.DB) *StockRepo { return &StockRepo{db: db} }

type ProductUpsertInput struct {
	Name              string
	Cost              float64
	OnHandQuantity    int
	FreeToUseQuantity int
	CategoryID        string
	LocationID        string
	Description       string
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
			p.on_hand_quantity,
			p.free_to_use_quantity,
			p.category_id,
			c.name AS category_name,
			p.location_id,
			l.name AS location_name,
			l.short_code AS location_short_code,
			COALESCE(array_agg(DISTINCT w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		JOIN locations.locations l ON l.id = p.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE (
			$1 = ''
			OR p.sku ILIKE '%' || $1 || '%'
			OR p.name ILIKE '%' || $1 || '%'
			OR c.name ILIKE '%' || $1 || '%'
			OR l.name ILIKE '%' || $1 || '%'
			OR COALESCE(p.description, '') ILIKE '%' || $1 || '%'
		)
		GROUP BY
			p.id, p.sku, p.name, p.cost, p.on_hand_quantity, p.free_to_use_quantity,
			p.category_id, c.name, p.location_id, l.name, l.short_code, p.description, p.updated_at
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
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *StockRepo) CreateProduct(input ProductUpsertInput) (*models.Product, error) {
	const q = `
		INSERT INTO stocks.products
			(name, cost, on_hand_quantity, free_to_use_quantity, category_id, location_id, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var productID string
	err := r.db.QueryRow(
		q,
		input.Name,
		input.Cost,
		input.OnHandQuantity,
		input.FreeToUseQuantity,
		input.CategoryID,
		input.LocationID,
		nullableText(input.Description),
	).Scan(&productID)
	if err != nil {
		return nil, err
	}

	return r.GetProductByID(productID)
}

func (r *StockRepo) UpdateProduct(productID string, input ProductUpsertInput) (*models.Product, error) {
	const q = `
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

	result, err := r.db.Exec(
		q,
		productID,
		input.Name,
		input.Cost,
		input.OnHandQuantity,
		input.FreeToUseQuantity,
		input.CategoryID,
		input.LocationID,
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
	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.cost,
			p.on_hand_quantity,
			p.free_to_use_quantity,
			p.category_id,
			c.name AS category_name,
			p.location_id,
			l.name AS location_name,
			l.short_code AS location_short_code,
			COALESCE(array_agg(DISTINCT w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			COALESCE(p.description, ''),
			p.updated_at
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		JOIN locations.locations l ON l.id = p.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE p.id = $1
		GROUP BY
			p.id, p.sku, p.name, p.cost, p.on_hand_quantity, p.free_to_use_quantity,
			p.category_id, c.name, p.location_id, l.name, l.short_code, p.description, p.updated_at`

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
	return &product, nil
}
