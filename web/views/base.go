package views

type StationView struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	EngName string `json:"name_en"`
}

type StationsView struct {
	Stations []StationView `json:"stations"`
}
