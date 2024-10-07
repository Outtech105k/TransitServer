package controllers

import (
	"database/sql"
	"errors"
	"fmt"

	"outtech105.com/transit_server/forms"
	"outtech105.com/transit_server/models"
)

var (
	ErrQueueEmpty = errors.New("queue is empty")
)

type Route struct {
	Operations  []models.Operation
	ViaStations map[uint]struct{} `json:"via_stations"`
}
type Routes []Route

func (r *Routes) enqueue(newRoute Route) {
	*r = append(*r, newRoute)
}

func (r *Routes) dequeue() (Route, error) {
	if len(*r) == 0 {
		return Route{}, ErrQueueEmpty
	}

	route := (*r)[0]
	*r = (*r)[1:]
	return route, nil
}

func (r *Routes) isEmpty() bool {
	return len(*r) == 0
}

func SearchTransit(req forms.TransitSearchForm, db *sql.DB) ([]Route, error) {
	reachedRoutes := make(Routes, 0, 10)

	firstOperations, err := models.SearchNextOperations(db, req.DepartStationID, *req.DepartDateTime)
	if err != nil {
		return []Route{}, fmt.Errorf("searchTransit: %w", err)
	}

	searchingRouteQueue := make(Routes, 0, 100)
	for _, operation := range firstOperations {
		newRoute := Route{
			Operations: []models.Operation{operation},
			ViaStations: map[uint]struct{}{
				operation.DepartStationID: {},
				operation.ArriveStationID: {},
			},
		}
		searchingRouteQueue.enqueue(newRoute)
	}

	// 続けて幅優先探索で先の経路を取得する
	for !searchingRouteQueue.isEmpty() {

		// TEMP: ルート長さが2に達したらやめる
		if len(searchingRouteQueue[0].Operations) == 20 {
			break
		}

		// 先頭の探索ルートを抜き出す
		lastRoute, err := searchingRouteQueue.dequeue()
		if err != nil {
			return []Route{}, fmt.Errorf("dequeueSearchingRouteQueue: %w", err)
		}

		// ルートが目的地に到達したら完成ルート一覧に追加
		if lastRoute.Operations[len(lastRoute.Operations)-1].ArriveStationID == req.ArriveStationID {
			reachedRoutes = append(reachedRoutes, lastRoute)
			continue
		}

		// 新たな探索を行い、発見数だけ延長する
		newOperations, err := models.SearchNextOperations(
			db,
			lastRoute.Operations[len(lastRoute.Operations)-1].ArriveStationID,
			lastRoute.Operations[len(lastRoute.Operations)-1].ArriveTime,
		)
		if err != nil {
			return []Route{}, fmt.Errorf("searchNextOperations: %w", err)
		}

		// 発見された移動に対し、経由駅に戻らない場合はenqueueする
		for _, newOperation := range newOperations {

			if _, isExists := lastRoute.ViaStations[newOperation.ArriveStationID]; isExists {
				continue
			}

			// Deep Copyをしてから新到達駅IDを追加
			newViaStations := make(map[uint]struct{})
			for viaStationID := range lastRoute.ViaStations {
				newViaStations[viaStationID] = struct{}{}
			}
			newViaStations[newOperation.ArriveStationID] = struct{}{}

			// Deep Copy をしてから新しい Operation を追加
			copiedLastOperations := make([]models.Operation, len(lastRoute.Operations))
			copy(copiedLastOperations, lastRoute.Operations)

			extendedRoute := Route{
				Operations:  append(copiedLastOperations, newOperation),
				ViaStations: newViaStations,
			}

			searchingRouteQueue.enqueue(extendedRoute)
		}
	}

	return reachedRoutes, nil
}
