package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"outtech105.com/transit_server/models"
	"outtech105.com/transit_server/views"
)

// 駅名キーワードから部分一致で駅を検索
func GetStationsByKeyword(db *sqlx.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		keyword := ctx.Query("keyword")
		if keyword == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Keyword must be specified."})
			return
		}

		stations, err := models.GetStationsByKeyword(db, keyword)
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

// 駅IDから駅情報を取得
func GetStationByID(db *sqlx.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "Invalid request."})
			return
		}

		station, err := models.GetStationByID(db, uint(id))
		if err != nil {
			if err == sql.ErrNoRows {
				ctx.AbortWithStatusJSON(http.StatusNotFound, views.ErrorView{Error: "Station not found."})
				return
			}

			ctx.AbortWithStatus(http.StatusInternalServerError)
			log.Printf("getStationByID: %s", err.Error())
			return
		}

		ctx.JSON(http.StatusOK, views.StationView(station))
	}
}
