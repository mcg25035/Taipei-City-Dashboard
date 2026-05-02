// Package controllers stores all the controllers for the Gin router.
package controllers

import (
	"fmt"
	"net/http"
	"regexp"

	"TaipeiCityDashboardBE/app/models"

	"github.com/gin-gonic/gin"
)

// validIdentifier matches PostgreSQL identifier shape we permit for table names.
// Limits the surface area against SQL injection: lowercase + digits + underscore,
// must start with a letter, length 1..63 (Postgres NAMEDATALEN-1).
var validIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// housekeepingProperties are columns we drop before serialising a row's
// properties to GeoJSON. They are either redundant (geometry is the geometry
// column itself), internal (ogc_fid is the row PK), or trigger-managed
// timestamps that the FE never displays.
var housekeepingProperties = []string{"wkb_geometry", "geometry", "ogc_fid", "_ctime", "_mtime"}

/*
GetGeojsonByIndex serves a FeatureCollection assembled directly from a
ready_data PostGIS table. Used by FE map_config rows whose `source` is
`be_geojson`. The path parameter `:index` MUST equal both the
`component_maps.index` value and the underlying PostGIS table name.

GET /api/v1/geojson/:index
*/
func GetGeojsonByIndex(c *gin.Context) {
	index := c.Param("index")

	if !validIdentifier.MatchString(index) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid index",
		})
		return
	}

	// Whitelist: only layers explicitly registered as be_geojson are exposed.
	// Querying an arbitrary index that happens to be a table name is rejected.
	var count int64
	if err := models.DBManager.Table("component_maps").
		Where("index = ? AND source = ?", index, "be_geojson").
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "no be_geojson layer registered for this index",
		})
		return
	}

	// Build the housekeeping minus expression: `to_jsonb(t) - 'col1' - 'col2' - ...`
	propertiesExpr := "to_jsonb(t)"
	for _, col := range housekeepingProperties {
		propertiesExpr += fmt.Sprintf(" - '%s'", col)
	}

	// Identifier is whitelisted by validIdentifier + component_maps lookup, so
	// it is safe to interpolate into the table reference here.
	// ST_AsGeoJSON precision 6 = ~11cm at the equator — well below visual
	// resolution of the FE map. Default is 9, which inflates payload with
	// digits that never affect rendering.
	sql := fmt.Sprintf(`
		SELECT json_build_object(
			'type', 'FeatureCollection',
			'features', COALESCE(json_agg(
				json_build_object(
					'type', 'Feature',
					'geometry', ST_AsGeoJSON(wkb_geometry, 6)::json,
					'properties', %s
				)
			), '[]'::json)
		)::text AS fc
		FROM public.%s AS t
		WHERE wkb_geometry IS NOT NULL
	`, propertiesExpr, index)

	var fc string
	if err := models.DBDashboard.Raw(sql).Scan(&fc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Serve as raw JSON. Disable caching so the FE always sees the latest
	// snapshot the DAG has loaded into PostGIS.
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Data(http.StatusOK, "application/json", []byte(fc))
}
