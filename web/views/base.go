package views

import "time"

// エラー時レスポンス
type ErrorView struct {
	Error string `json:"error"`
}

// models.Stationに対応
type StationView struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	EngName string `json:"name_en"`
}

type StationsView struct {
	Stations []StationView `json:"stations"`
}

// models.Operationに対応
type OperationView struct {
	TrainID         uint      `json:"train_id"`
	Order           uint      `json:"order"`
	DepartStationID uint      `json:"depart_station_id"`
	DepartDatetime  time.Time `json:"depart_datetime"`
	ArriveStationID uint      `json:"arrive_station_id"`
	ArriveDatetime  time.Time `json:"arrive_datetime"`
}
