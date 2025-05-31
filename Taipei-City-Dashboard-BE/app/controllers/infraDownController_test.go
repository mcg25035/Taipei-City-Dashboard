package controllers_test

import (
	"Taipei-City-Dashboard-BE/app/controllers"
	"Taipei-City-Dashboard-BE/app/models"
	"Taipei-City-Dashboard-BE/global"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Re-use testDB from homeDownController_test.go or declare locally if preferred.
// For simplicity in this example, we assume it might be run in a context where testDB is shared
// or re-declared. If running tests in parallel or isolation, ensure DB setup is distinct.
// var testDB *gorm.DB // If not already declared in the package (e.g. from another _test.go file)

func setupInfraDownTestRouter() *gin.Engine {
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}
	global.DB = testDB

	err = global.DB.AutoMigrate(&models.InfraDownEvent{})
	if err != nil {
		panic(fmt.Sprintf("Failed to migrate database schema: %v", err))
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		infraDownRoutes := v1.Group("/infraDown")
		{
			infraDownRoutes.POST("", controllers.CreateInfraDownEvent)
			infraDownRoutes.GET("", controllers.GetInfraDownEvents)
			infraDownRoutes.DELETE("/:id", controllers.DeleteInfraDownEvent)
		}
	}
	return router
}

func teardownInfraDownTestData() {
	if testDB != nil {
		testDB.Exec("DELETE FROM infra_down_events")
	}
}

func TestCreateInfraDownEvent_Success(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	timeMin := time.Now().Unix()
	timeMax := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf(`{"type": "scheduled_maintenance", "timeMin": %d, "timeMax": %d, "data": ["server_room_A", "network_switch_B"]}`, timeMin, timeMax)

	req, _ := http.NewRequest("POST", "/api/v1/infraDown", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response struct {
		Message string          `json:"message"`
		Data    map[string]uint `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Event created successfully", response.Message)
	assert.Contains(t, response.Data, "id")
	eventId := response.Data["id"]
	assert.NotZero(t, eventId)

	var eventInDB models.InfraDownEvent
	result := global.DB.First(&eventInDB, eventId)
	assert.NoError(t, result.Error)
	assert.Equal(t, "scheduled_maintenance", eventInDB.Type)
	assert.Equal(t, time.Unix(timeMin, 0).UTC(), eventInDB.StartTime.UTC()) // Compare UTC times
	assert.Equal(t, time.Unix(timeMax, 0).UTC(), eventInDB.EndTime.UTC())
	assert.Equal(t, pq.StringArray{"server_room_A", "network_switch_B"}, eventInDB.AreaData)
	assert.NotZero(t, eventInDB.CreatedAt)
}

func TestCreateInfraDownEvent_InvalidInput(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	testCases := []struct {
		name    string
		payload string
		message string
	}{
		{
			name:    "Missing type",
			payload: fmt.Sprintf(`{"timeMin": %d, "timeMax": %d, "data": ["area"]}`, time.Now().Unix(), time.Now().Add(1*time.Hour).Unix()),
			message: "Invalid input",
		},
		{
			name:    "Missing timeMin",
			payload: fmt.Sprintf(`{"type": "outage", "timeMax": %d, "data": ["area"]}`, time.Now().Add(1*time.Hour).Unix()),
			message: "Invalid input",
		},
		{
			name:    "Missing timeMax",
			payload: fmt.Sprintf(`{"type": "outage", "timeMin": %d, "data": ["area"]}`, time.Now().Unix()),
			message: "Invalid input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/infraDown", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			var errorResponse GenericResponse // Assuming GenericResponse is defined (as in homeDownController_test)
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse.Message, tc.message)
		})
	}
}

func TestGetInfraDownEvents_Empty(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	req, _ := http.NewRequest("GET", "/api/v1/infraDown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response GenericResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Events retrieved successfully", response.Message)
	var events []models.InfraDownEvent
	json.Unmarshal(response.Data, &events)
	assert.Len(t, events, 0)
}

func seedInfraEvents(db *gorm.DB, numEvents int) []models.InfraDownEvent {
	events := make([]models.InfraDownEvent, numEvents)
	baseTime := time.Now()
	for i := 0; i < numEvents; i++ {
		event := models.InfraDownEvent{
			Type:      fmt.Sprintf("type%d", i%2), // type0, type1, type0 ...
			StartTime: baseTime.Add(time.Duration(i) * time.Hour),
			EndTime:   baseTime.Add(time.Duration(i+1) * time.Hour),
			AreaData:  pq.StringArray{fmt.Sprintf("area%d", i)},
			CreatedAt: baseTime.Add(-time.Duration(i) * time.Minute), // Ensure different created_at for ordering
		}
		db.Create(&event)
		events[i] = event
	}
	return events
}

func TestGetInfraDownEvents_WithDataAndFilters(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	seededEvents := seedInfraEvents(global.DB, 5) // type0, type1, type0, type1, type0

	// Test Case 1: No filters
	req, _ := http.NewRequest("GET", "/api/v1/infraDown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp1 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp1)
	var events1 []models.InfraDownEvent
	json.Unmarshal(resp1.Data, &events1)
	assert.Len(t, events1, 5)
	// Check descending order of created_at
	for i := 0; i < len(events1)-1; i++ {
		assert.True(t, events1[i].CreatedAt.After(events1[i+1].CreatedAt) || events1[i].CreatedAt.Equal(events1[i+1].CreatedAt))
	}

	// Test Case 2: Filter by type "type0"
	req, _ = http.NewRequest("GET", "/api/v1/infraDown?type=type0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp2 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp2)
	var events2 []models.InfraDownEvent
	json.Unmarshal(resp2.Data, &events2)
	assert.Len(t, events2, 3)
	for _, e := range events2 {
		assert.Equal(t, "type0", e.Type)
	}

	// Test Case 3: Filter by timeMin
	// seededEvents[2].StartTime is the reference
	timeMinUnix := seededEvents[2].StartTime.Unix()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/infraDown?timeMin=%d", timeMinUnix), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp3 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp3)
	var events3 []models.InfraDownEvent
	json.Unmarshal(resp3.Data, &events3)
	assert.Len(t, events3, 3) // events[2], events[3], events[4]
	for _, e := range events3 {
		assert.True(t, e.StartTime.Unix() >= timeMinUnix || e.StartTime.Equal(time.Unix(timeMinUnix,0)))
	}

	// Test Case 4: Filter by timeMax
	// seededEvents[2].EndTime is the reference
	timeMaxUnix := seededEvents[2].EndTime.Unix()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/infraDown?timeMax=%d", timeMaxUnix), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp4 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp4)
	var events4 []models.InfraDownEvent
	json.Unmarshal(resp4.Data, &events4)
	assert.Len(t, events4, 3) // events[0], events[1], events[2]
	for _, e := range events4 {
		assert.True(t, e.EndTime.Unix() <= timeMaxUnix || e.EndTime.Equal(time.Unix(timeMaxUnix,0)))
	}

	// Test Case 5: Combine type and timeMin
	timeMinUnixCombined := seededEvents[1].StartTime.Unix() // Start time of the first "type1" event
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/infraDown?type=type1&timeMin=%d", timeMinUnixCombined), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp5 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp5)
	var events5 []models.InfraDownEvent
	json.Unmarshal(resp5.Data, &events5) // Should be events[1] and events[3]
	assert.Len(t, events5, 2)
	for _, e := range events5 {
		assert.Equal(t, "type1", e.Type)
		assert.True(t, e.StartTime.Unix() >= timeMinUnixCombined)
	}
}

func TestDeleteInfraDownEvent_Success(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	events := seedInfraEvents(global.DB, 1)
	eventToDelete := events[0]

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/infraDown/%d", eventToDelete.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response GenericResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Event deleted successfully", response.Message)

	var eventInDB models.InfraDownEvent
	result := global.DB.First(&eventInDB, eventToDelete.ID)
	assert.Error(t, result.Error) // Should be gorm.ErrRecordNotFound
	assert.True(t, result.Error == gorm.ErrRecordNotFound)
}

func TestDeleteInfraDownEvent_NotFound(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	nonExistentID := uint(9999)
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/infraDown/%d", nonExistentID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errorResponse GenericResponse
	json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.Equal(t, "Event not found", errorResponse.Message)
}

func TestDeleteInfraDownEvent_InvalidID(t *testing.T) {
	router := setupInfraDownTestRouter()
	defer teardownInfraDownTestData()

	req, _ := http.NewRequest("DELETE", "/api/v1/infraDown/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errorResponse GenericResponse
	json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.Equal(t, "Invalid ID format", errorResponse.Message)
}
```
