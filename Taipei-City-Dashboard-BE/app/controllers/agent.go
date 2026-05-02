// Package controllers stores all the controllers for the Gin router.
package controllers

import (
	"net/http"
	"strconv"

	"TaipeiCityDashboardBE/app/models"
	"TaipeiCityDashboardBE/app/util"

	"github.com/gin-gonic/gin"
)

/*
Agent-facing endpoints. Designed for AI agent / LLM tool-use:
  - JSON in / JSON out
  - Single round trip per logical operation
  - Stable response shape with explicit `status` field
*/

type agentSearchRequest struct {
	Query string  `json:"query"`
	Limit int     `json:"limit"`
	Score float64 `json:"score"`
}

/*
AgentSearchComponents performs natural-language search over public components.
POST /api/v1/agent/search

Body:
  {"query": "交通壅塞", "limit": 10, "score": 0.78}

Response:
  {"status": "success", "data": [CityComponentScore, ...]}
*/
func AgentSearchComponents(c *gin.Context) {
	var req agentSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "query is required"})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 30 {
		req.Limit = 30
	}
	if req.Score < 0 || req.Score > 1 {
		req.Score = 0.78
	}

	results, err := models.GetComponentByQueryVector(req.Query, req.Limit, req.Score)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": results})
}

/*
AgentGetComponentData returns full component config plus its chart data in one call.
GET /api/v1/agent/component/:id?city=taipei&timefrom=...&timeto=...

Response:
  {
    "status": "success",
    "component": CityComponent,
    "query_type": "two_d" | "three_d" | "percent" | "time" | "map_legend" | "",
    "chart": ... | null,
    "categories": [...]   // only when query_type is three_d / percent
  }
*/
func AgentGetComponentData(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid component ID"})
		return
	}

	city := c.DefaultQuery("city", "taipei")
	if !(city == "taipei" || city == "metrotaipei") {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid City Name"})
		return
	}

	component, err := models.GetComponentByID(id, city)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "component not found"})
		return
	}

	queryType, queryString, err := models.GetComponentChartDataQuery(id, city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	timeFrom, timeTo, err := util.GetTime(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	resp := gin.H{
		"status":     "success",
		"component":  component,
		"query_type": queryType,
		"chart":      nil,
	}

	if queryString == "" || queryType == "" {
		c.JSON(http.StatusOK, resp)
		return
	}

	switch queryType {
	case "two_d":
		chartData, err := models.GetTwoDimensionalData(&queryString, timeFrom, timeTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		resp["chart"] = chartData
	case "three_d", "percent":
		chartData, categories, err := models.GetThreeDimensionalData(&queryString, timeFrom, timeTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		resp["chart"] = chartData
		resp["categories"] = categories
	case "time":
		chartData, err := models.GetTimeSeriesData(&queryString, timeFrom, timeTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		resp["chart"] = chartData
	case "map_legend":
		chartData, err := models.GetMapLegendData(&queryString, timeFrom, timeTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		resp["chart"] = chartData
	}

	c.JSON(http.StatusOK, resp)
}

type agentParkingBBoxRequest struct {
	LatMin float64 `json:"lat_min" form:"lat_min"`
	LatMax float64 `json:"lat_max" form:"lat_max"`
	LngMin float64 `json:"lng_min" form:"lng_min"`
	LngMax float64 `json:"lng_max" form:"lng_max"`
}

/*
AgentParkingByBBox returns parking lot count + average occupancy inside a
lng/lat bounding box across both Taipei and NewTaipei realtime tables.
Lots reporting no realtime data (occupied_rate = -99) are counted as fully
occupied (1.0) for the average — explicit agent contract.

POST /api/v1/agent/parking-bbox
Body or query string:

	{"lat_min": 25.01, "lat_max": 25.10, "lng_min": 121.50, "lng_max": 121.60}

Response:

	{
	  "status": "success",
	  "bbox": {...echoed input...},
	  "data": {
	    "total_lots": N,
	    "with_realtime": M,
	    "occupied_rate_avg": 0.78,
	    "by_city": [
	      {"city": "taipei",    "lots": ..., "with_realtime": ..., "occupied_rate_avg": ...},
	      {"city": "newtaipei", "lots": ..., "with_realtime": ..., "occupied_rate_avg": ...}
	    ]
	  }
	}
*/
func AgentParkingByBBox(c *gin.Context) {
	var req agentParkingBBoxRequest
	// Accept either JSON body or query-string params.
	_ = c.ShouldBindJSON(&req)
	if req.LngMin == 0 && req.LngMax == 0 && req.LatMin == 0 && req.LatMax == 0 {
		_ = c.ShouldBindQuery(&req)
	}

	res, err := models.GetParkingByBBox(req.LngMin, req.LatMin, req.LngMax, req.LatMax)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"bbox":   req,
		"data":   res,
	})
}

type agentYoubikeBBoxRequest struct {
	LatMin float64 `json:"lat_min" form:"lat_min"`
	LatMax float64 `json:"lat_max" form:"lat_max"`
	LngMin float64 `json:"lng_min" form:"lng_min"`
	LngMax float64 `json:"lng_max" form:"lng_max"`
}

/*
AgentYoubikeByBBox returns every YouBike station whose location falls inside
the given lng/lat bounding box, joined to its latest realtime availability
(tran_ubike_realtime for Taipei, tran_ubike_realtime_new_tpe for New Taipei).
Stations with no realtime row come back with null availability fields.

POST /api/v1/agent/youbike-bbox
GET  /api/v1/agent/youbike-bbox?lat_min=...&lat_max=...&lng_min=...&lng_max=...

Body or query string:

	{"lat_min": 25.01, "lat_max": 25.10, "lng_min": 121.50, "lng_max": 121.60}

Response:

	{
	  "status": "success",
	  "bbox":   {...echoed input...},
	  "count":  N,
	  "data": [
	    {
	      "station_uid": "...", "station_id": "...",
	      "name": "...", "addr": "...",
	      "lng": 121.5, "lat": 25.05, "county": "Taipei",
	      "service_type": "UBike2.0", "bike_capacity": 28,
	      "service_status": "正常營運",
	      "available_rent_general_bikes": 5,
	      "available_rent_electric_bikes": 0,
	      "available_return_bikes": 23,
	      "data_time": "2025-02-19T03:19:14+00:00"
	    }, ...
	  ]
	}
*/
func AgentYoubikeByBBox(c *gin.Context) {
	var req agentYoubikeBBoxRequest
	// Accept either JSON body or query-string params.
	_ = c.ShouldBindJSON(&req)
	if req.LngMin == 0 && req.LngMax == 0 && req.LatMin == 0 && req.LatMax == 0 {
		_ = c.ShouldBindQuery(&req)
	}

	stations, err := models.GetYoubikeStationsByBBox(req.LngMin, req.LatMin, req.LngMax, req.LatMax)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"bbox":   req,
		"count":  len(stations),
		"data":   stations,
	})
}
