package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"outtech105.com/transit_server/models"
	"outtech105.com/transit_server/views"
)

// 駅IDから駅情報を取得
func GetStationByID(db *sql.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "invalid Request."})
			return
		}

		station, err := models.GetStationByID(db, uint(id))
		if err != nil {
			if err == sql.ErrNoRows {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "station Not Found."})
				return
			}

			ctx.AbortWithStatus(http.StatusInternalServerError)
			log.Printf("getStationByID: %s", err.Error())
			return
		}

		ctx.JSON(http.StatusOK, views.StationView(station))
	}
}

// 駅名から検索
func GetStationsByKeyword(db *sql.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		stations, err := models.GetStationsByKeyword(db, ctx.Query("keyword"))
		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			log.Printf("getStationByKeyword: %s", err.Error())
			return
		}

		stationsView := make([]views.StationView, 0, len(stations))
		for _, sta := range stations {
			stationsView = append(stationsView, views.StationView(sta))
		}
		ctx.JSON(http.StatusOK, views.StationsView{Stations: stationsView})
	}
}
