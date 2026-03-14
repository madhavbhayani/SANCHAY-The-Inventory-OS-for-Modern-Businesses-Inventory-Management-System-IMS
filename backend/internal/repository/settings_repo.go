package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"

	"github.com/lib/pq"
)

var (
	ErrWarehouseShortCodeTaken = errors.New("warehouse short code already exists")
	ErrLocationShortCodeTaken  = errors.New("location short code already exists")
)

type SettingsRepo struct{ db *sql.DB }

func NewSettingsRepo(db *sql.DB) *SettingsRepo { return &SettingsRepo{db: db} }

func (r *SettingsRepo) CreateWarehouse(name, shortCode, address, description string) (*models.Warehouse, error) {
	const q = `
		INSERT INTO locations.warehouses (name, short_code, address, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, short_code, COALESCE(address, ''), COALESCE(description, ''), created_at`

	var w models.Warehouse
	err := r.db.QueryRow(q, name, shortCode, nullableText(address), nullableText(description)).Scan(
		&w.ID, &w.Name, &w.ShortCode, &w.Address, &w.Description, &w.CreatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "warehouses_short_code") {
			return nil, ErrWarehouseShortCodeTaken
		}
		return nil, err
	}
	return &w, nil
}

func (r *SettingsRepo) ListWarehouses() ([]models.Warehouse, error) {
	const q = `
		SELECT id, name, short_code, COALESCE(address, ''), COALESCE(description, ''), created_at
		FROM locations.warehouses
		ORDER BY created_at DESC`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	warehouses := make([]models.Warehouse, 0)
	for rows.Next() {
		var w models.Warehouse
		if err := rows.Scan(&w.ID, &w.Name, &w.ShortCode, &w.Address, &w.Description, &w.CreatedAt); err != nil {
			return nil, err
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, rows.Err()
}

func (r *SettingsRepo) CreateLocation(name, shortCode string, warehouseIDs []string) (*models.Location, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const insertLocation = `
		INSERT INTO locations.locations (name, short_code)
		VALUES ($1, $2)
		RETURNING id, created_at`

	var location models.Location
	location.Name = name
	location.ShortCode = shortCode

	if err := tx.QueryRow(insertLocation, name, shortCode).Scan(&location.ID, &location.CreatedAt); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "locations_short_code") {
			return nil, ErrLocationShortCodeTaken
		}
		return nil, err
	}

	const insertLink = `
		INSERT INTO locations.location_warehouses (location_id, warehouse_id)
		VALUES ($1, $2)`

	location.WarehouseIDs = make([]string, 0, len(warehouseIDs))
	for _, wid := range warehouseIDs {
		wid = strings.TrimSpace(wid)
		if wid == "" {
			continue
		}
		if _, err := tx.Exec(insertLink, location.ID, wid); err != nil {
			return nil, fmt.Errorf("failed linking location to warehouse: %w", err)
		}
		location.WarehouseIDs = append(location.WarehouseIDs, wid)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *SettingsRepo) ListLocations() ([]models.Location, error) {
	const q = `
		SELECT
			l.id,
			l.name,
			l.short_code,
			COALESCE(array_agg(lw.warehouse_id::text ORDER BY lw.warehouse_id)
				FILTER (WHERE lw.warehouse_id IS NOT NULL), '{}') AS warehouse_ids,
			COALESCE(array_agg(w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			l.created_at
		FROM locations.locations l
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		GROUP BY l.id, l.name, l.short_code, l.created_at
		ORDER BY l.created_at DESC`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]models.Location, 0)
	for rows.Next() {
		var loc models.Location
		var warehouseIDs, warehouseNames pq.StringArray
		if err := rows.Scan(
			&loc.ID,
			&loc.Name,
			&loc.ShortCode,
			&warehouseIDs,
			&warehouseNames,
			&loc.CreatedAt,
		); err != nil {
			return nil, err
		}
		loc.WarehouseIDs = append([]string(nil), warehouseIDs...)
		loc.WarehouseNames = append([]string(nil), warehouseNames...)
		locations = append(locations, loc)
	}
	return locations, rows.Err()
}

func nullableText(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
