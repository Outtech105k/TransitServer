package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"outtech105.com/transit_server/models"
	"outtech105.com/transit_server/views"
)

// 駅IDから駅情報を取得
func GetStation(db *sql.DB) func(*gin.Context) {
	return func(ctx *gin.Context) {
		id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "invalid Request."})
			return
		}

		station, err := models.GetStationWithID(db, uint(id))
		if err != nil {
			if err == sql.ErrNoRows {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, views.ErrorView{Error: "station Not Found."})
				return
			}

			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		ctx.JSON(http.StatusOK, views.StationView(station))
	}
}
