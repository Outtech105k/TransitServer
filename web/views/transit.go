package views

import "time"

// APIリクエストのレスポンス型

type ErrorView struct {
	Error string `json:"error"`
}

type OperationView struct {
	TrainID         uint      `json:"train_id"`
	Order           uint      `json:"order"`
	DepartStationID uint      `json:"depart_station_id"`
	DepartTime      time.Time `json:"depart_time"`
	ArriveStationID uint      `json:"arrive_station_id"`
	ArriveTime      time.Time `json:"arrive_time"`
}

type TransitSearchView struct {
	Routes []RouteView `json:"routes"`
}

type RouteView struct {
	Operations []OperationView `json:"operations"`
}

type StationView struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	EngName string `json:"name_en"`
}
