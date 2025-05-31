package controllers

import (
	"net/http"
	"strconv"
	"time"

	"TaipeiCityDashboardBE/app/models"
	"TaipeiCityDashboardBE/app/util"
	"TaipeiCityDashboardBE/global"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateInfraDownEvent godoc
// @Summary Create a new infra down event
// @Description Add a new infra down event announcement
// @Tags infraDown
// @Accept  json
// @Produce  json
// @Param   event_data body models.InfraDownEvent true "Infra Down Event Data (timeMin, timeMax will be set to current time and current time + 1 hour)"
// @Success 201 {object} util.Response{data=map[string]uint} "Successfully created, returns id"
// @Failure 400 {object} util.Response "Invalid input"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/infraDown [post]
func CreateInfraDownEvent(c *gin.Context) {
	var input struct {
		Type     string `json:"type" binding:"required"`
		TimeMin  int64  `json:"timeMin"` // Unix timestamp, no longer required
		TimeMax  int64  `json:"timeMax"` // Unix timestamp, no longer required
		AreaData string `json:"data"`    // AreaData is now directly a string
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.ResponseError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Set startTime and endTime to current time and current time + 1 hour
	startTime := time.Now()
	endTime := time.Now().Add(time.Hour)

	event := models.InfraDownEvent{
		Type:      input.Type,
		StartTime: startTime,
		EndTime:   endTime,
		AreaData:  input.AreaData, // Directly assign string
	}

	if err := global.DB.Create(&event).Error; err != nil {
		util.ResponseError(c, http.StatusInternalServerError, "Failed to create event: "+err.Error())
		return
	}

	util.ResponseSuccess(c, http.StatusCreated, "Event created successfully", gin.H{"id": event.ID})
}

// GetInfraDownEvents godoc
// @Summary Get all infra down events
// @Description Get a list of all infra down events without pagination or filters
// @Tags infraDown
// @Accept  json
// @Produce  json
// @Success 200 {object} util.Response{data=[]models.InfraDownEvent} "Successfully retrieved events"
// @Failure 500 {object} util.Response "Internal server error"
// @Router /api/v1/infraDown [get]
func GetInfraDownEvents(c *gin.Context) {

	var events []models.InfraDownEvent
	query := global.DB.Model(&models.InfraDownEvent{})

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
