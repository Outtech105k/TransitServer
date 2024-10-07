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

	// 出発時刻が設定された場合の探索
	if req.DepartDateTime != nil {
		// 出発駅から発車する直近列車を取得
		firstOperations, err := models.SearchNextDepartOperations(db, req.DepartStationID, *req.DepartDateTime)
		if err != nil {
			return []Route{}, fmt.Errorf("searchTransit: %w", err)
		}

		// 取得結果からルートを生成
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

			// 最後に到達した駅・時刻を基準に新たな探索
			newOperations, err := models.SearchNextDepartOperations(
				db,
				lastRoute.Operations[len(lastRoute.Operations)-1].ArriveStationID,
				lastRoute.Operations[len(lastRoute.Operations)-1].ArriveTime,
			)
			if err != nil {
				return []Route{}, fmt.Errorf("searchNextOperations: %w", err)
			}

			// 発見された移動について、適切なものを探索キューに追加
			for _, newOperation := range newOperations {
				// 発見された移動に対し、経由駅に戻る移動は除外する
				if _, isExists := lastRoute.ViaStations[newOperation.ArriveStationID]; isExists {
					continue
				}

				// 経由駅集合のDeep Copyをしてから新到達駅IDを追加
				newViaStations := make(map[uint]struct{})
				for viaStationID := range lastRoute.ViaStations {
					newViaStations[viaStationID] = struct{}{}
				}
				newViaStations[newOperation.ArriveStationID] = struct{}{}

				// 探索済みルートのDeep Copy をして、新たなルートオブジェクトを生成
				copiedLastOperations := make([]models.Operation, len(lastRoute.Operations))
				copy(copiedLastOperations, lastRoute.Operations)

				// ルートをEnqueue
				extendedRoute := Route{
					Operations:  append(copiedLastOperations, newOperation),
					ViaStations: newViaStations,
				}
				searchingRouteQueue.enqueue(extendedRoute)
			}
		}

		// 目的地に到達したルートのみ返す
		return reachedRoutes, nil

	}

	return []Route{}, fmt.Errorf("not implemented")
}
