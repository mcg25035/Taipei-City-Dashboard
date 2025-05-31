package controllers

import (
	"net/http"
	"strconv"

	"TaipeiCityDashboardBE/app/models"
	"TaipeiCityDashboardBE/app/util" // Assuming 'util' contains common response functions
	"TaipeiCityDashboardBE/global"   // Assuming 'global' contains the DB instance

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateHomeDownEvent godoc
// @Summary Create a new home down event
// @Description Add a new home down event reported by a user
// @Tags homeDown
// @Accept  json
// @Produce  json
// @Param   event_data body models.HomeDownEvent true "Home Down Event Data"
// @Success 201 {object} util.Response{data=models.HomeDownEvent} "Successfully created"
// @Failure 400 {object} util.Response "Invalid input"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/homeDown [post]
func CreateHomeDownEvent(c *gin.Context) {
	var input struct {
		Where []float64 `json:"where" binding:"required,len=2"` // [longitude, latitude]
		Type  string    `json:"type" binding:"required"`
		Name  *string   `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.ResponseError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	event := models.HomeDownEvent{
		Longitude: input.Where[0],
		Latitude:  input.Where[1],
		Type:      input.Type,
		Name:      input.Name,
	}

	// logs.FInfo("Attempting to create HomeDownEvent. global.DB: %v, event: %+v", global.DB, event)
	if err := global.DB.Create(&event).Error; err != nil {
		// logs.FError("Failed to create HomeDownEvent. Error: %v", err)
		util.ResponseError(c, http.StatusInternalServerError, "Failed to create event: "+err.Error())
		return
	}
	// logs.FInfo("HomeDownEvent created successfully. Event ID: %d", event.ID)

	util.ResponseSuccess(c, http.StatusCreated, "Event created successfully", event)
}

// GetHomeDownEvents godoc
// @Summary Get all home down events
// @Description Get a list of home down events with optional filters and pagination
// @Tags homeDown
// @Accept  json
// @Produce  json
// @Param   tab query int false "Page number (default 1, 50 items per page)"
// @Param   type query string false "Filter by event type"
// @Param   LatitudeMin query number false "Minimum latitude"
// @Param   LatitudeMax query number false "Maximum latitude"
// @Param   LongitudeMin query number false "Minimum longitude"
// @Param   LongitudeMax query number false "Maximum longitude"
// @Success 200 {object} util.Response{data=[]models.HomeDownEvent} "Successfully retrieved events"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/homeDown [get]
func GetHomeDownEvents(c *gin.Context) {
	var events []models.HomeDownEvent
	query := global.DB.Model(&models.HomeDownEvent{})

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("tab", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 50
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// Type filter
	if eventType := c.Query("type"); eventType != "" {
		query = query.Where("type = ?", eventType)
	}

	// Latitude filter
	latMinStr := c.Query("LatitudeMin")
	latMaxStr := c.Query("LatitudeMax")
	if latMinStr != "" && latMaxStr != "" {
		latMin, errMin := strconv.ParseFloat(latMinStr, 64)
		latMax, errMax := strconv.ParseFloat(latMaxStr, 64)
		if errMin == nil && errMax == nil {
			query = query.Where("latitude BETWEEN ? AND ?", latMin, latMax)
		}
	}

	// Longitude filter
	lonMinStr := c.Query("LongitudeMin")
	lonMaxStr := c.Query("LongitudeMax")
	if lonMinStr != "" && lonMaxStr != "" {
		lonMin, errMin := strconv.ParseFloat(lonMinStr, 64)
		lonMax, errMax := strconv.ParseFloat(lonMaxStr, 64)
		if errMin == nil && errMax == nil {
			query = query.Where("longitude BETWEEN ? AND ?", lonMin, lonMax)
		}
	}

	query = query.Order("reported_at DESC")

	if err := query.Find(&events).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			util.ResponseSuccess(c, http.StatusOK, "No events found", []models.HomeDownEvent{})
			return
		}
		util.ResponseError(c, http.StatusInternalServerError, "Failed to retrieve events: "+err.Error())
		return
	}

	util.ResponseSuccess(c, http.StatusOK, "Events retrieved successfully", events)
}
