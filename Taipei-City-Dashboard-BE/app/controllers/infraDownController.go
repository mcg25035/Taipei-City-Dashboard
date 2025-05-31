package controllers

import (
	"net/http"
	"strconv"
	"time"

	"Taipei-City-Dashboard-BE/app/models"
	"Taipei-City-Dashboard-BE/app/util"
	"Taipei-City-Dashboard-BE/global"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq" // For pq.StringArray if used for JSONB
	"gorm.io/gorm"
)

// CreateInfraDownEvent godoc
// @Summary Create a new infra down event
// @Description Add a new infra down event announcement
// @Tags infraDown
// @Accept  json
// @Produce  json
// @Param   event_data body models.InfraDownEvent true "Infra Down Event Data (timeMin, timeMax as Unix timestamps)"
// @Success 201 {object} util.Response{data=map[string]uint} "Successfully created, returns id"
// @Failure 400 {object} util.Response "Invalid input"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/infraDown [post]
func CreateInfraDownEvent(c *gin.Context) {
	var input struct {
		Type     string   `json:"type" binding:"required"`
		TimeMin  int64    `json:"timeMin" binding:"required"` // Unix timestamp
		TimeMax  int64    `json:"timeMax" binding:"required"` // Unix timestamp
		AreaData []string `json:"data"`                       // Assuming JSON array of strings for simplicity
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.ResponseError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Convert Unix timestamps to time.Time
	startTime := time.Unix(input.TimeMin, 0)
	endTime := time.Unix(input.TimeMax, 0)

	event := models.InfraDownEvent{
		Type:      input.Type,
		StartTime: startTime,
		EndTime:   endTime,
		AreaData:  pq.StringArray(input.AreaData), // Ensure this matches the model's AreaData type
	}

	if err := global.DB.Create(&event).Error; err != nil {
		util.ResponseError(c, http.StatusInternalServerError, "Failed to create event: "+err.Error())
		return
	}

	util.ResponseSuccess(c, http.StatusCreated, "Event created successfully", gin.H{"id": event.ID})
}

// GetInfraDownEvents godoc
// @Summary Get all infra down events
// @Description Get a list of infra down events with optional filters and pagination
// @Tags infraDown
// @Accept  json
// @Produce  json
// @Param   tab query int false "Page number (default 1, 50 items per page)"
// @Param   type query string false "Filter by event type"
// @Param   timeMin query int false "Minimum start time (Unix timestamp)"
// @Param   timeMax query int false "Maximum end time (Unix timestamp)"
// @Param   data query string false "Filter by area data (JSON string, exact match or specific query logic needed)"
// @Success 200 {object} util.Response{data=[]models.InfraDownEvent} "Successfully retrieved events"
// @Failure 400 {object} util.Response "Invalid input for timeMin/timeMax"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/infraDown [get]
func GetInfraDownEvents(c *gin.Context) {
	var events []models.InfraDownEvent
	query := global.DB.Model(&models.InfraDownEvent{})

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

	// TimeMin filter
	if timeMinStr := c.Query("timeMin"); timeMinStr != "" {
		timeMinUnix, err := strconv.ParseInt(timeMinStr, 10, 64)
		if err != nil {
			util.ResponseError(c, http.StatusBadRequest, "Invalid timeMin format: must be a Unix timestamp")
			return
		}
		query = query.Where("start_time >= ?", time.Unix(timeMinUnix, 0))
	}

	// TimeMax filter
	if timeMaxStr := c.Query("timeMax"); timeMaxStr != "" {
		timeMaxUnix, err := strconv.ParseInt(timeMaxStr, 10, 64)
		if err != nil {
			util.ResponseError(c, http.StatusBadRequest, "Invalid timeMax format: must be a Unix timestamp")
			return
		}
		query = query.Where("end_time <= ?", time.Unix(timeMaxUnix, 0))
	}

	// Data filter (simple exact match for pq.StringArray, might need more complex JSON query for other types)
	// Note: Filtering JSONB effectively often requires specific database functions (e.g., @>, ?, ?&, ?|)
	// This example provides a placeholder for how one might start, but it's not a comprehensive JSON query solution.
	// The issue states "阿龍定義" (Ah Long defines it), so the exact query logic for 'data' might be complex.
	// For now, this will be a placeholder or a very simple match if possible.
	// if dataQuery := c.Query("data"); dataQuery != "" {
	//    // This is tricky with pq.StringArray directly for partial matches.
	//    // If AreaData were datatypes.JSON, you could use GORM's JSON querying features.
	//    // For pq.StringArray, you might need raw SQL or a different approach for complex queries.
	//    // Example: query = query.Where("area_data @> ?", dataQuery) // If dataQuery is a valid JSONB array string for containment
	// }

	query = query.Order("created_at DESC")

	if err := query.Find(&events).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			util.ResponseSuccess(c, http.StatusOK, "No events found", []models.InfraDownEvent{})
			return
		}
		util.ResponseError(c, http.StatusInternalServerError, "Failed to retrieve events: "+err.Error())
		return
	}

	util.ResponseSuccess(c, http.StatusOK, "Events retrieved successfully", events)
}

// DeleteInfraDownEvent godoc
// @Summary Delete an infra down event
// @Description Delete an infra down event by its ID
// @Tags infraDown
// @Accept  json
// @Produce  json
// @Param   id path int true "Event ID"
// @Success 200 {object} util.Response "Successfully deleted"
// @Failure 400 {object} util.Response "Invalid ID format"
// @Failure 404 {object} util.Response "Event not found"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/infraDown/{id} [delete]
func DeleteInfraDownEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.ResponseError(c, http.StatusBadRequest, "Invalid ID format")
		return
	}

	result := global.DB.Delete(&models.InfraDownEvent{}, uint(id))
	if result.Error != nil {
		util.ResponseError(c, http.StatusInternalServerError, "Failed to delete event: "+result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		util.ResponseError(c, http.StatusNotFound, "Event not found")
		return
	}

	util.ResponseSuccess(c, http.StatusOK, "Event deleted successfully", nil)
}
