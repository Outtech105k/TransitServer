package models

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Station struct {
	ID      uint
	Name    string
	EngName string
}

func GetStationByID(db *sqlx.DB, id uint) (Station, error) {
	var station Station
	err := db.QueryRow(
		`SELECT id, name, name_en FROM stations WHERE id = ?`,
		id,
	).Scan(&station.ID, &station.Name, &station.EngName)
	return station, err
}

func GetStationsByName(db *sqlx.DB, name string) ([]Station, error) {
	stations := make([]Station, 0, 10)
	query := `
SELECT id, name, name_en FROM stations
WHERE name = ?
`
	rows, err := db.Query(query, name)
	if err != nil {
		return nil, fmt.Errorf("executeQuery: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Station
		if err := rows.Scan(&s.ID, &s.Name, &s.EngName); err != nil {
			return nil, fmt.Errorf("scanRecord: %w", err)
		}
		stations = append(stations, s)
	}

	return stations, nil
}
func GetStationsByKeyword(db *sqlx.DB, keyword string) ([]Station, error) {
	stations := make([]Station, 0, 10)
	keywordWithQuery := fmt.Sprintf("%%%s%%", keyword)
	query := `
SELECT id, name, name_en FROM stations
WHERE name LIKE ? OR name_en LIKE ?
`
	rows, err := db.Query(query, keywordWithQuery, keywordWithQuery)
	if err != nil {
		return nil, fmt.Errorf("executeQuery: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Station
		if err := rows.Scan(&s.ID, &s.Name, &s.EngName); err != nil {
			return nil, fmt.Errorf("scanRecord: %w", err)
		}
		stations = append(stations, s)
	}

	return stations, nil
}
