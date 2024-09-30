package handler

import (
	"database/sql"
	"errors"
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
	TrainID         uint   `json:"train_id"`
	Order           uint   `json:"order"`
	DepartStationID uint   `json:"depart_station_id"`
	DepartTime      string `json:"depart_time"`
	ArriveStationID uint   `json:"arrive_station_id"`
	ArriveTime      string `json:"arrive_time"`
}

type Route []Operation
type Routes []Route

func (routes *Routes) enqueue(new_route Route) {
	*routes = append(*routes, new_route)
}

func (routes *Routes) dequeue() (Route, error) {
	if len(*routes) == 0 {
		return Route{}, ErrQueueEmpty
	}
	route := (*routes)[0]
	*routes = (*routes)[1:]
	return route, nil
}

func (routes *Routes) isEmpty() bool {
	return len(*routes) == 0
}

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

	routes, err := searchTransit(request, h.DB)
	if err != nil {
		log.Printf("Error searching transit: %v", err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": routes})
}

func searchTransit(req TransitSearchRequest, db *sql.DB) ([]Route, error) {
	searchingRoutes := make(Routes, 0, 100)
	reachedRoutes := make(Routes, 0, 10)
	// ルート起点を検索
	operations, err := searchNextOperations(db, req.DepartStationID, *req.DepartDateTime)
	if err != nil {
		return []Route{}, err
	}

	for _, op := range operations {
		searchingRoutes.enqueue(Route{op})
	}

	// ルート続きを検索
	cnt := 0
	for !searchingRoutes.isEmpty() && cnt < 1000000 {
		// 目的地に到達したルートを排除
		route, _ := searchingRoutes.dequeue()
		if route[len(route)-1].ArriveStationID == req.ArriveStationID {
			reachedRoutes = append(reachedRoutes, route)
		} else {
			parsedTime, err := time.Parse("15:04:05", route[len(route)-1].ArriveTime)
			if err != nil {
				return nil, err
			}

			operations, err := searchNextOperations(
				db,
				route[len(route)-1].ArriveStationID,
				time.Date(
					req.DepartDateTime.Year(),
					req.DepartDateTime.Month(),
					req.DepartDateTime.Day(),
					parsedTime.Hour(),
					parsedTime.Minute(),
					parsedTime.Second(),
					parsedTime.Nanosecond(),
					time.Local,
				),
			)
			if err != nil {
				return nil, err
			}

			for _, op := range operations {
				searchingRoutes.enqueue(append(route, op))
			}
		}
		cnt++
	}

	return reachedRoutes, nil
}

// 入力バリデーション(実行中止時 -> false)
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
func searchNextOperations(db *sql.DB, departStationID uint, departTime time.Time) ([]Operation, error) {
	rows, err := db.Query(`
SELECT train_id, op_order, dep_sta_id, dep_time, arr_sta_id, arr_time
FROM operations
WHERE dep_sta_id = ?
AND dep_time >= ?
ORDER BY dep_time
`,
		departStationID,
		departTime.Format("15:04:05"),
	)
	if err != nil {
		return []Operation{}, err
	}

	operations := make([]Operation, 0, 5)
	var op Operation
	for rows.Next() {
		err := rows.Scan(&op.TrainID, &op.Order, &op.DepartStationID, &op.DepartTime, &op.ArriveStationID, &op.ArriveTime)
		if err != nil {
			return []Operation{}, err
		}
		operations = append(operations, op)
	}

	return operations, nil
}
