package forms

import "time"

type TransitSearchForm struct {
	DepartStationID uint       `json:"depart_station_id" binding:"required"`
	DepartDateTime  *time.Time `json:"depart_datetime"`
	ArriveStationID uint       `json:"arrive_station_id" binding:"required"`
	ArriveDateTime  *time.Time `json:"arrive_datetime"`
}
