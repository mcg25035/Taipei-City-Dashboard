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
