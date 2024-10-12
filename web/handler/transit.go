package handler

import (
	"database/sql"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"outtech105.com/transit_server/controllers"
	"outtech105.com/transit_server/forms"
	"outtech105.com/transit_server/models"
	"outtech105.com/transit_server/views"
)

// 乗り換え案内を行うハンドラーを返す
func SearchTransitHandler(db *sql.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		// リクエストJSONの必要事項解析
		var request forms.TransitSearchForm
		if err := ctx.ShouldBindJSON(&request); err != nil {
			log.Printf("Error binding JSON in SearchTransit: %v", err)
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Parameters are missing."})
			return
		}

		// 時刻設定が出発・到着の片方のみであるか (XOR)
		if (request.DepartDateTime == nil) == (request.ArriveDateTime == nil) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Either the departure time or the arrival time must be set, but not both."})
			return
		}

		// 出発駅指定が、ID/名前の片方のみであるか (XOR)
		if (request.DepartStationID == nil) == (request.DepartStationName == nil) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Eithor the departure station id or the departure station name must be set, but not both."})
			return
		}

		// 到着駅指定が、ID/名前の片方のみであるか (XOR)
		if (request.ArriveStationID == nil) == (request.ArriveStationName == nil) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Eithor the arrive station id or the departure arrive name must be set, but not both."})
			return
		}

		// TODO: 出発時刻設定限定(Remove it future)
		if request.ArriveDateTime != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Only Depart Time Setting (on maintenance)."})
			return
		}

		// 出発駅の解析
		if request.DepartStationName != nil {
			stationCandidates, err := models.GetStationsByName(db, *request.DepartStationName)
			if err != nil {
				ctx.AbortWithStatus(http.StatusInternalServerError)
				log.Printf("Error getting depart station by name: %v", err)
				return
			}
			if len(stationCandidates) != 1 {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "The departure station name entered does not uniquely identify the station. You should specify a station name that can be identified."})
				return
			}
			request.DepartStationID = &stationCandidates[0].ID
		}

		// 到着駅の解析
		if request.ArriveStationName != nil {
			stationCandidates, err := models.GetStationsByName(db, *request.ArriveStationName)
			if err != nil {
				ctx.AbortWithStatus(http.StatusInternalServerError)
				log.Printf("Error getting arrive station by name: %v", err)
				return
			}
			if len(stationCandidates) != 1 {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "The arrive station name entered does not uniquely identify the station. You should specify a station name that can be identified."})
				return
			}
			request.ArriveStationID = &stationCandidates[0].ID
		}

		// 出発・到着駅IDが存在するか
		if err := models.CheckExistsStationIDs(db, *request.DepartStationID, *request.ArriveStationID); err != nil {
			if err == models.ErrStationIDsMissing {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: models.ErrStationIDsMissing.Error()})
			} else {
				log.Print("checkExistsStationIDs: %w", err)
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		// 出発・到着駅IDが異なるか
		if request.DepartStationID == request.ArriveStationID {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Departure station ID and arrival station ID must be different."})
			return
		}

		// 読み込んだ時刻をJSTに設定
		jst, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			log.Printf("Error loading location: %v", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if request.DepartDateTime != nil {
			*request.DepartDateTime = (*request.DepartDateTime).In(jst)
		}
		if request.ArriveDateTime != nil {
			*request.ArriveDateTime = (*request.ArriveDateTime).In(jst)
		}

		// 出発時刻を中心に乗換探索
		routes, err := controllers.SearchTransitByDepart(
			controllers.TransitSearchByDepart{
				DepartStationID: *request.DepartStationID,
				DepartDateTime:  *request.DepartDateTime,
				ArriveStationID: *request.ArriveStationID,
			},
			db,
		)
		if err != nil {
			log.Printf("Error searching transit: %v", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// 到着時刻順にソート
		sort.SliceStable(routes, func(i, j int) bool {
			return routes[i].Operations[len(routes[i].Operations)-1].ArriveTime.Before(
				routes[j].Operations[len(routes[j].Operations)-1].ArriveTime,
			)
		})

		// 結果を5件以下に制限
		routes = routes[0:min(len(routes), 5)]

		// 検索結果をレスポンス構造体に代入
		viaStationsSet := make(map[uint]struct{})
		routesView := make([]views.RouteView, len(routes))
		for i, route := range routes {
			operationsView := make([]views.OperationView, len(route.Operations))
			for j, operation := range route.Operations {
				operationsView[j] = views.OperationView(operation)
				viaStationsSet[operation.DepartStationID] = struct{}{}
				viaStationsSet[operation.ArriveStationID] = struct{}{}
			}
			routesView[i] = views.RouteView{
				Operations: operationsView,
			}
		}

		viaStationsView := make([]views.StationView, 0, len(viaStationsSet))
		for id := range viaStationsSet {
			station, err := models.GetStationByID(db, id)
			if err != nil {
				ctx.AbortWithStatus(http.StatusInternalServerError)
				log.Printf("get station with ID: %s", err.Error())
				return
			}
			viaStationsView = append(viaStationsView, views.StationView(station))
		}

		sort.SliceStable(viaStationsView, func(i, j int) bool {
			return viaStationsView[i].ID < viaStationsView[j].ID
		})

		// 検索結果リクエストを返却
		ctx.JSON(http.StatusOK, views.TransitSearchView{
			Stations: viaStationsView,
			Routes:   routesView,
		})
	}
}
