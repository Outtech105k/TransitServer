package handler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type TransitSearchRequest struct {
	DepartStationID uint       `json:"depart_station_id" binding:"required"`
	DepartDateTime  *time.Time `json:"depart_datetime"`
	ArriveStationID uint       `json:"arrive_station_id" binding:"required"`
	ArriveDateTime  *time.Time `json:"arrive_datetime"`
}

type Operation struct {
	TrainID         uint
	Order           uint
	DepartStationID uint
	DepartTime      string
	ArriveStationID uint
	ArriveTime      string
}

type DBHandler struct {
	DB *sql.DB
}

func NewDBHandler(db *sql.DB) *DBHandler {
	return &DBHandler{DB: db}
}

func (h *DBHandler) SearchNextStations() error {
	rows, err := h.DB.Query(`
SELECT o1.train_id, o1.op_order, o1.dep_sta_id, o1.dep_time, o1.arr_sta_id, o1.arr_time
FROM operations o1 
INNER JOIN (
	SELECT dep_sta_id, MIN(dep_time) AS next_dep_time, arr_sta_id
	FROM operations
	WHERE dep_sta_id = 3
	GROUP BY dep_sta_id, arr_sta_id
	ORDER BY dep_sta_id ASC
) o2
ON o1.dep_sta_id = o2.dep_sta_id
AND o1.dep_time = o2.next_dep_time
AND o1.arr_sta_id = o2.arr_sta_id
`)
	if err != nil {
		return err
	}

	var op Operation
	for rows.Next() {
		err := rows.Scan(&op.TrainID, &op.Order, &op.DepartStationID, &op.DepartTime, &op.ArriveStationID, &op.ArriveTime)
		if err != nil {
			return err
		}
		fmt.Printf("%+v\n", op)
	}

	return nil
}

func (h *DBHandler) SearchTransit(ctx *gin.Context) {
	var request TransitSearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("Error binding JSON in SearchTransit: %v", err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	ctx.JSON(http.StatusOK, request)
}
