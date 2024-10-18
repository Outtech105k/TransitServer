package forms

import "time"

// 乗換案内探索のリクエストフォーマット
type TransitSearchForm struct {
	DepartStationName *string    `json:"depart_station_name"`
	DepartStationID   *uint      `json:"depart_station_id"`
	DepartDateTime    *time.Time `json:"depart_datetime"`
	ArriveStationName *string    `json:"arrive_station_name"`
	ArriveStationID   *uint      `json:"arrive_station_id"`
	ArriveDateTime    *time.Time `json:"arrive_datetime"`
}
