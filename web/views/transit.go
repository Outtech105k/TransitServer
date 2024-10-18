package views

// 乗換案内情報のレスポンス型

type TransitSearchView struct {
	Stations []StationView `json:"stations"`
	Routes   []RouteView   `json:"routes"`
}

type RouteView struct {
	Operations []OperationView `json:"operations"`
}
