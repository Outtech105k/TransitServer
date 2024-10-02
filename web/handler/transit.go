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
	TrainID         uint   `json:"train_id"`
	Order           uint   `json:"order"`
	DepartStationID uint   `json:"depart_station_id"`
	DepartTime      string `json:"depart_time"`
	ArriveStationID uint   `json:"arrive_station_id"`
	ArriveTime      string `json:"arrive_time"`
}

type Route struct {
	Operations  []Operation
	ViaStations map[uint]struct{}
}
type Routes []Route

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
	firstOperations, err := searchNextOperations(db, req.DepartStationID, *req.DepartDateTime)
	if err != nil {
		return []Route{}, fmt.Errorf("searchTransit: %w", err)
	}

	reachedRoutes := make([]Route, 0, 100)
	for _, operation := range firstOperations {
		reachedRoutes = append(reachedRoutes, Route{
			Operations: []Operation{operation},
		})
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
