package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"outtech105.com/transit_server/models"
)

var (
	ErrQueueEmpty = errors.New("queue is empty")
)

type TransitSearchByDepart struct {
	DepartStationID uint
	DepartDateTime  time.Time
	ArriveStationID uint
}

type Route struct {
	Operations  []models.Operation `json:"operations"`
	ViaStations map[uint]struct{}  `json:"via_stations"` // 経由した駅の集合
}

// RouteのQueue構造とMethods
type Routes []Route // RouteのQueue構造のための型
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

// 列車の乗り換え案内を検索(出発時刻基準)
func SearchTransitByDepart(req TransitSearchByDepart, db *sql.DB) ([]Route, error) {
	reachedRoutes := make(Routes, 0, 10)

	// 出発駅から発車する直近列車を取得
	firstOperations, err := models.SearchNextDepartOperations(db, req.DepartStationID, req.DepartDateTime)
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
	// 追加探査すべきルートが無くなるまで続ける(データ取得に問題がなければ、有限時間で終了する)
	for !searchingRouteQueue.isEmpty() {
		// 先頭の探索ルートを抜き出す
		lastRoute, err := searchingRouteQueue.dequeue()
		if err != nil {
			return []Route{}, fmt.Errorf("dequeueSearchingRouteQueue: %w", err)
		}

		// 生成されたルートオブジェクトが目的地に到達していれば、完成ルートリストに追加
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
			// 経由駅に戻る移動は除外する
			if _, isExists := lastRoute.ViaStations[newOperation.ArriveStationID]; isExists {
				continue
			}

			// 経由駅集合のDeep Copyをしてから新到達駅IDを追加
			newViaStations := make(map[uint]struct{})
			for viaStationID := range lastRoute.ViaStations {
				newViaStations[viaStationID] = struct{}{}
			}
			newViaStations[newOperation.ArriveStationID] = struct{}{}

			// 探索済みルートのDeep Copyをして、新たなルートオブジェクトを生成
			copiedLastOperations := make([]models.Operation, len(lastRoute.Operations), len(lastRoute.Operations)+1)
			copy(copiedLastOperations, lastRoute.Operations)

			extendedRoute := Route{
				Operations:  append(copiedLastOperations, newOperation),
				ViaStations: newViaStations,
			}

			// 目的地に到達していないので、ルートをEnqueue
			searchingRouteQueue.enqueue(extendedRoute)
		}
	}

	// 目的地に到達したルートのみ返す
	return reachedRoutes, nil
}
