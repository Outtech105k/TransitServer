package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrStationIDsMissing = errors.New("invalid station ID")
	ErrQueueEmpty        = errors.New("queue is empty")
)

type TransitSearchRequest struct {
	DepartStationID uint       `json:"depart_station_id" binding:"required"`
	DepartDateTime  *time.Time `json:"depart_datetime"`
	ArriveStationID uint       `json:"arrive_station_id" binding:"required"`
	ArriveDateTime  *time.Time `json:"arrive_datetime"`
}

type Operation struct {
	TrainID         uint      `json:"train_id"`
	Order           uint      `json:"order"`
	DepartStationID uint      `json:"depart_station_id"`
	DepartTime      time.Time `json:"depart_time"`
	ArriveStationID uint      `json:"arrive_station_id"`
	ArriveTime      time.Time `json:"arrive_time"`
}

type Route struct {
	Operations  []Operation
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

type TransitSearchResponse struct {
	Routes []RouteResponse `json:"routes"`
}

type RouteResponse []Operation

type DBHandler struct {
	DB *sql.DB
}

func NewDBHandler(db *sql.DB) *DBHandler {
	return &DBHandler{DB: db}
}

func (h *DBHandler) SearchTransitHandle(ctx *gin.Context) {
	var request TransitSearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("Error binding JSON in SearchTransit: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Parameters are missing."})
		return
	}

	if !validateRequest(ctx, request, h.DB) {
		return
	}

	// 時刻をJSTに設定
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

	routes, err := searchTransit(request, h.DB)
	if err != nil {
		log.Printf("Error searching transit: %v", err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	routesResponse := make([]RouteResponse, len(routes))
	for i, v := range routes {
		routesResponse[i] = v.Operations
	}

	ctx.JSON(http.StatusOK, TransitSearchResponse{
		Routes: routesResponse,
	})
}

func searchTransit(req TransitSearchRequest, db *sql.DB) ([]Route, error) {
	reachedRoutes := make(Routes, 0, 10)

	firstOperations, err := searchNextOperations(db, req.DepartStationID, *req.DepartDateTime)
	if err != nil {
		return []Route{}, fmt.Errorf("searchTransit: %w", err)
	}

	searchingRouteQueue := make(Routes, 0, 100)
	for _, operation := range firstOperations {
		newRoute := Route{
			Operations: []Operation{operation},
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
			return []Route{}, err
		}

		// ルートが目的地に到達したら完成ルート一覧に追加
		if lastRoute.Operations[len(lastRoute.Operations)-1].ArriveStationID == req.ArriveStationID {
			reachedRoutes = append(reachedRoutes, lastRoute)
			continue
		}

		// 新たな探索を行い、発見数だけ延長する
		newOperations, err := searchNextOperations(
			db,
			lastRoute.Operations[len(lastRoute.Operations)-1].ArriveStationID,
			lastRoute.Operations[len(lastRoute.Operations)-1].ArriveTime,
		)
		if err != nil {
			return []Route{}, err
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
			copiedLastOperations := make([]Operation, len(lastRoute.Operations))
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

// 入力バリデーション(不合格時: false)
func validateRequest(ctx *gin.Context, request TransitSearchRequest, db *sql.DB) bool {
	// 時刻設定が出発・到着の片方のみであるか (XOR)
	if (request.DepartDateTime == nil) == (request.ArriveDateTime == nil) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Either the departure time or the arrival time must be set, but not both."})
		return false
	}

	// 出発・到着駅IDが異なるか
	if request.DepartStationID == request.ArriveStationID {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Departure station ID and arrival station ID must be different."})
		return false
	}

	// 出発・到着駅IDが存在するか
	if err := checkExistsStationIDs(db, request.DepartStationID, request.ArriveStationID); err != nil {
		if err == ErrStationIDsMissing {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": ErrStationIDsMissing.Error()})
		} else {
			log.Print("checkExistsStationIDs: %w", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
		return false
	}

	// TODO: 出発時刻設定限定(Remove it future)
	if request.ArriveDateTime != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Only Depart Time Setting (on maintenance)."})
		return false
	}

	return true
}

func checkExistsStationIDs(db *sql.DB, depID, arrID uint) error {
	var result bool
	err := db.QueryRow(`SELECT COUNT(*) = 2 FROM stations WHERE id IN (?, ?);`, depID, arrID).Scan(&result)
	if err != nil {
		return err
	}

	if !result {
		return ErrStationIDsMissing
	} else {
		return nil
	}
}

// 指定駅から指定時間以降に発車する列車を取得
// TODO: 00:00を超えて走行する列車の日付更新
// TODO: 日付を超えた乗り換え案内
func searchNextOperations(db *sql.DB, departStationID uint, fastestDepartDateTime time.Time) ([]Operation, error) {
	rows, err := db.Query(`
SELECT o1.train_id, o1.op_order, o1.dep_sta_id, o1.dep_time, o1.arr_sta_id, o1.arr_time
FROM operations o1 
INNER JOIN (
	SELECT dep_sta_id, MIN(dep_time) AS next_dep_time, arr_sta_id
	FROM operations
	WHERE dep_sta_id = ? AND dep_time >= ?
	GROUP BY dep_sta_id, arr_sta_id
	ORDER BY dep_sta_id ASC
) o2
ON o1.dep_sta_id = o2.dep_sta_id
AND o1.dep_time = o2.next_dep_time
AND o1.arr_sta_id = o2.arr_sta_id
`,
		departStationID,
		fastestDepartDateTime.Format("15:04:05"),
	)
	if err != nil {
		return []Operation{}, err
	}

	operations := make([]Operation, 0, 5)
	var (
		op               Operation
		departTimeString string
		arriveTimeString string
	)

	for rows.Next() {
		err := rows.Scan(&op.TrainID, &op.Order, &op.DepartStationID, &departTimeString, &op.ArriveStationID, &arriveTimeString)
		if err != nil {
			return []Operation{}, err
		}

		op.DepartTime, err = updateTimeWithString(fastestDepartDateTime, departTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateDepartTimeString: %w", err)
		}

		op.ArriveTime, err = updateTimeWithString(fastestDepartDateTime, arriveTimeString)
		if err != nil {
			return []Operation{}, fmt.Errorf("updateArriveTimeString: %w", err)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

func updateTimeWithString(originalTime time.Time, timeString string) (time.Time, error) {
	// "15:04:05"形式の時刻部分をパース
	parsedTime, err := time.Parse("15:04:05", timeString)
	if err != nil {
		return time.Time{}, err
	}

	// 年月日を維持しつつ、時刻を上書き
	updatedTime := time.Date(
		originalTime.Year(),
		originalTime.Month(),
		originalTime.Day(),
		parsedTime.Hour(),       // 時刻部分を置き換え
		parsedTime.Minute(),     // 分部分を置き換え
		parsedTime.Second(),     // 秒部分を置き換え
		0,                       // ナノ秒は0に設定
		originalTime.Location(), // タイムゾーンも元のものを使用
	)

	return updatedTime, nil
}
