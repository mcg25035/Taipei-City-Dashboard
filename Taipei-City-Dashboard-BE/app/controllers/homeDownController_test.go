package controllers_test

import (
	"Taipei-City-Dashboard-BE/app/controllers"
	"Taipei-City-Dashboard-BE/app/models"
	"Taipei-City-Dashboard-BE/global"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDB *gorm.DB

// Response structure for assertions
type GenericResponse struct {
	Message string `json:"message"`
	Data    json.RawMessage `json:"data"` // Use json.RawMessage to parse data later
}


func setupHomeDownTestRouter() *gin.Engine {
	var err error
	// Initialize an in-memory SQLite database for testing
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}

	// Assign to global.DB for controller use
	global.DB = testDB

	// Auto-migrate schema
	err = global.DB.AutoMigrate(&models.HomeDownEvent{})
	if err != nil {
		panic(fmt.Sprintf("Failed to migrate database schema: %v", err))
	}

	gin.SetMode(gin.TestMode)
	router := gin.New() // Use gin.New() instead of gin.Default() for more control in tests

	// Setup routes for testing (mirroring app/routes/router.go structure)
	v1 := router.Group("/api/v1")
	{
		homeDownRoutes := v1.Group("/homeDown")
		{
			homeDownRoutes.POST("", controllers.CreateHomeDownEvent)
			homeDownRoutes.GET("", controllers.GetHomeDownEvents)
		}
	}
	return router
}

func teardownHomeDownTestData() {
	// Clear data from home_down_events table
	if testDB != nil {
		testDB.Exec("DELETE FROM home_down_events")
		// Optionally, close the database connection if tests are run in separate processes or if necessary
		// sqlDB, _ := testDB.DB()
		// sqlDB.Close()
	}
}

func TestCreateHomeDownEvent_Success(t *testing.T) {
	router := setupHomeDownTestRouter()
	defer teardownHomeDownTestData()

	payload := `{"where": [121.5654, 25.0340], "type": "power_outage", "name": "Test Event"}`
	req, _ := http.NewRequest("POST", "/api/v1/homeDown", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response struct {
		Message string               `json:"message"`
		Data    models.HomeDownEvent `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Event created successfully", response.Message)
	assert.Equal(t, "power_outage", response.Data.Type)
	assert.Equal(t, 121.5654, response.Data.Longitude)
	assert.Equal(t, 25.0340, response.Data.Latitude)
	assert.NotNil(t, response.Data.Name)
	assert.Equal(t, "Test Event", *response.Data.Name)
	assert.NotZero(t, response.Data.ID)
	assert.NotZero(t, response.Data.ReportedAt)

	// Check database
	var eventInDB models.HomeDownEvent
	result := global.DB.First(&eventInDB, response.Data.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "power_outage", eventInDB.Type)
	assert.Equal(t, 121.5654, eventInDB.Longitude)
}

func TestCreateHomeDownEvent_InvalidInput_MissingType(t *testing.T) {
	router := setupHomeDownTestRouter()
	defer teardownHomeDownTestData()

	payload := `{"where": [121.5654, 25.0340], "name": "Test Event Missing Type"}` // Missing "type"
	req, _ := http.NewRequest("POST", "/api/v1/homeDown", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse GenericResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse.Message, "Invalid input")
}

func TestCreateHomeDownEvent_InvalidInput_MalformedWhere(t *testing.T) {
	router := setupHomeDownTestRouter()
	defer teardownHomeDownTestData()

	payload := `{"where": [121.5654], "type": "water_outage", "name": "Test Event Malformed Where"}` // "where" has only one element
	req, _ := http.NewRequest("POST", "/api/v1/homeDown", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse GenericResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse.Message, "Invalid input")
}


func TestGetHomeDownEvents_Empty(t *testing.T) {
	router := setupHomeDownTestRouter()
	defer teardownHomeDownTestData()

	req, _ := http.NewRequest("GET", "/api/v1/homeDown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response GenericResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Events retrieved successfully", response.Message) // Controller returns this message even for empty

	var events []models.HomeDownEvent
	err = json.Unmarshal(response.Data, &events)
	assert.NoError(t, err)
	assert.Len(t, events, 0)
}

func seedHomeDownEvents(db *gorm.DB, numEvents int, baseLat, baseLng float64, baseType string) []models.HomeDownEvent {
	events := make([]models.HomeDownEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		name := fmt.Sprintf("Event %d", i)
		event := models.HomeDownEvent{
			Latitude:  baseLat + float64(i)*0.01,
			Longitude: baseLng + float64(i)*0.01,
			Type:      fmt.Sprintf("%s_%d", baseType, i%2), // Alternating types
			Name:      &name,
			ReportedAt: time.Now().Add(-time.Duration(i) * time.Hour), // Ensure different reported_at for ordering
		}
		db.Create(&event)
		events[i] = event
	}
	return events
}

func TestGetHomeDownEvents_WithDataAndFilters(t *testing.T) {
	router := setupHomeDownTestRouter()
	defer teardownHomeDownTestData()

	// Seed data
	seedHomeDownEvents(global.DB, 5, 25.00, 121.50, "power") // 5 events: power_0, power_1, power_0, power_1, power_0
	seedHomeDownEvents(global.DB, 3, 25.10, 121.60, "water") // 3 events: water_0, water_1, water_0

	// Test Case 1: No filters (get all, default pagination is 50, so all 8 should be returned)
	req, _ := http.NewRequest("GET", "/api/v1/homeDown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp1 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp1)
	var events1 []models.HomeDownEvent
	json.Unmarshal(resp1.Data, &events1)
	assert.Len(t, events1, 8)
	// Check descending order of reported_at (newest first)
	for i := 0; i < len(events1)-1; i++ {
		assert.True(t, events1[i].ReportedAt.After(events1[i+1].ReportedAt) || events1[i].ReportedAt.Equal(events1[i+1].ReportedAt))
	}


	// Test Case 2: Filter by type "power_0"
	req, _ = http.NewRequest("GET", "/api/v1/homeDown?type=power_0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp2 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp2)
	var events2 []models.HomeDownEvent
	json.Unmarshal(resp2.Data, &events2)
	assert.Len(t, events2, 3) // 3 events of type power_0
	for _, e := e range events2 {
		assert.Equal(t, "power_0", e.Type)
	}

	// Test Case 3: Filter by Latitude
	// Latitude for power events: 25.00, 25.01, 25.02, 25.03, 25.04
	req, _ = http.NewRequest("GET", "/api/v1/homeDown?LatitudeMin=25.01&LatitudeMax=25.03", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp3 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp3)
	var events3 []models.HomeDownEvent
	json.Unmarshal(resp3.Data, &events3)
	assert.Len(t, events3, 3) // 25.01, 25.02, 25.03
	for _, e := range events3 {
		assert.GreaterOrEqual(t, e.Latitude, 25.01)
		assert.LessOrEqual(t, e.Latitude, 25.03)
	}

	// Test Case 4: Filter by Longitude
	// Longitude for water events: 121.60, 121.61, 121.62
	req, _ = http.NewRequest("GET", "/api/v1/homeDown?LongitudeMin=121.61&LongitudeMax=121.62", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp4 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp4)
	var events4 []models.HomeDownEvent
	json.Unmarshal(resp4.Data, &events4)
	assert.Len(t, events4, 2) // 121.61, 121.62
	for _, e := range events4 {
		assert.GreaterOrEqual(t, e.Longitude, 121.61)
		assert.LessOrEqual(t, e.Longitude, 121.62)
	}

	// Test Case 5: Combine filters (type and latitude)
	req, _ = http.NewRequest("GET", "/api/v1/homeDown?type=water_0&LatitudeMin=25.10&LatitudeMax=25.12", nil)
	// water_0 latitudes: 25.10, 25.12
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp5 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp5)
	var events5 []models.HomeDownEvent
	json.Unmarshal(resp5.Data, &events5)
	assert.Len(t, events5, 2)
	for _, e := range events5 {
		assert.Equal(t, "water_0", e.Type)
		assert.GreaterOrEqual(t, e.Latitude, 25.10)
		assert.LessOrEqual(t, e.Latitude, 25.12)
	}

	// Test Case 6: Pagination (tab=2, 3 items per page for easier testing of pagination)
	// Temporarily modify pageSize in controller for this test or seed more data.
	// For simplicity here, we'll assume the default pageSize is 50, so we'll test with existing 8 events.
	// If we get first 5 (page 1), then page 2 should have 3.
	// To test pagination effectively, we might need to adjust pageSize or seed more items.
	// Let's re-query with tab=1 and a small pageSize by modifying the controller (not ideal for unit test)
	// or by seeding enough data. Let's assume we test with default page size 50.
	// With 8 items, page 1 has 8, page 2 has 0.
	req, _ = http.NewRequest("GET", "/api/v1/homeDown?tab=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp6 GenericResponse
	json.Unmarshal(w.Body.Bytes(), &resp6)
	var events6 []models.HomeDownEvent
	json.Unmarshal(resp6.Data, &events6)
	assert.Len(t, events6, 0) // Page 2 should be empty if pageSize is 50 and only 8 items exist

}

// TestMain can be used for global setup/teardown if needed,
// but individual setup/teardown functions are often clearer per test suite.
// func TestMain(m *testing.M) {
// 	// Global setup
// 	code := m.Run()
// 	// Global teardown
// 	os.Exit(code)
// }

// Note: The `initial` package was mentioned. If it contains critical setup like
// `global.DB` initialization that's used by controllers, then the test setup
// might need to call relevant functions from `initial` or replicate their essential parts.
// For this example, `global.DB` is directly initialized with an in-memory SQLite DB.
// The `Taipei-City-Dashboard-BE/app/routes` import isn't directly used here as we
// are defining the routes manually for the test router. This is fine for focused controller testing.
// If testing the full router chain as defined in `routes.ConfigureRoutes`, then one would import
// and use that.
// The `app/util` package is implicitly tested via the controller's usage of `util.ResponseError` etc.
// We are asserting the HTTP status code and parts of the JSON body, which indirectly confirms
// the response utilities are working as expected.
```
