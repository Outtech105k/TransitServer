package handler

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"outtech105.com/transit_server/controllers"
	"outtech105.com/transit_server/forms"
	"outtech105.com/transit_server/models"
	"outtech105.com/transit_server/views"
)

func SearchTransitHandler(db *sql.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		var request forms.TransitSearchForm
		if err := c.ShouldBindJSON(&request); err != nil {
			log.Printf("Error binding JSON in SearchTransit: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Parameters are missing."})
			return
		}

		if !validateRequest(c, request, db) {
			return
		}

		// 時刻をJSTに設定
		jst, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			log.Printf("Error loading location: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if request.DepartDateTime != nil {
			*request.DepartDateTime = (*request.DepartDateTime).In(jst)
		}
		if request.ArriveDateTime != nil {
			*request.ArriveDateTime = (*request.ArriveDateTime).In(jst)
		}

		// 出発時刻を中心に乗換探索
		routes, err := controllers.SearchTransit(request, db)
		if err != nil {
			log.Printf("Error searching transit: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		routesView := make([]views.RouteView, len(routes))
		for i, route := range routes {
			operationsView := make([]views.OperationView, len(route.Operations))
			for j, ope := range route.Operations {
				operationsView[j] = views.OperationView(ope)
			}
			routesView[i] = views.RouteView{
				Operations: operationsView,
			}
		}

		c.JSON(http.StatusOK, views.TransitSearchView{
			Routes: routesView,
		})
	}
}

// 入力バリデーション(不合格時: false)
func validateRequest(ctx *gin.Context, request forms.TransitSearchForm, db *sql.DB) bool {
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
	if err := models.CheckExistsStationIDs(db, request.DepartStationID, request.ArriveStationID); err != nil {
		if err == models.ErrStationIDsMissing {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": models.ErrStationIDsMissing.Error()})
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
