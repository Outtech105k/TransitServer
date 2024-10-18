package models

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBのstationsスキーマに対応
type Station struct {
	ID      uint   `db:"id"`
	Name    string `db:"name"`
	EngName string `db:"name_en"`
}

// 駅IDからDB問い合わせをし、駅情報を返す
func GetStationByID(db *sqlx.DB, id uint) (Station, error) {
	var station Station
	err := db.QueryRowx(
		`SELECT id, name, name_en FROM stations WHERE id = ?`,
		id,
	).StructScan(&station)
	return station, err
}

// 駅名から完全一致検索で駅一覧を返す
func GetStationsByName(db *sqlx.DB, name string) ([]Station, error) {
	stations := make([]Station, 0, 10)
	query := `
SELECT id, name, name_en FROM stations
WHERE name = ?
`
	rows, err := db.Queryx(query, name)
	if err != nil {
		return nil, fmt.Errorf("executeQuery: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Station
		if err := rows.StructScan(&s); err != nil {
			return nil, fmt.Errorf("scanRecord: %w", err)
		}
		stations = append(stations, s)
	}

	return stations, nil
}

// キーワードから部分一致検索で駅一覧を返す
func GetStationsByKeyword(db *sqlx.DB, keyword string) ([]Station, error) {
	stations := make([]Station, 0, 10)
	keywordWithQuery := fmt.Sprintf("%%%s%%", keyword)
	query := `
SELECT id, name, name_en FROM stations
WHERE name LIKE ? OR name_en LIKE ?
`
	rows, err := db.Queryx(query, keywordWithQuery, keywordWithQuery)
	if err != nil {
		return nil, fmt.Errorf("executeQuery: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Station
		if err := rows.StructScan(&s); err != nil {
			return nil, fmt.Errorf("scanRecord: %w", err)
		}
		stations = append(stations, s)
	}

	return stations, nil
}
