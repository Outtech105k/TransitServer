package models

import (
	"database/sql"
)

type Station struct {
	ID      uint
	Name    string
	EngName string
}

func GetStationWithID(db *sql.DB, id uint) (Station, error) {
	var station Station
	err := db.QueryRow(
		`SELECT id, name, name_en FROM stations WHERE id = ?`,
		id,
	).Scan(&station.ID, &station.Name, &station.EngName)
	return station, err
}
