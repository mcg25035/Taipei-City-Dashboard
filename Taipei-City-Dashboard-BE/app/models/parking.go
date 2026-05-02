// Package models — bbox-aggregated parking lot stats consumed by /api/v1/agent.
package models

import (
	"errors"
)

// ParkingBBoxCityStat holds aggregate stats per city within a bbox.
type ParkingBBoxCityStat struct {
	City             string  `json:"city"`               // "taipei" | "newtaipei"
	Lots             int64   `json:"lots"`               // # of parking lots inside bbox
	WithRealtime     int64   `json:"with_realtime"`      // # of lots that report realtime occupancy
	OccupiedRateAvg  float64 `json:"occupied_rate_avg"`  // avg occupancy [0,1], -99 treated as 1.0
}

// ParkingBBoxResult is the full response payload.
type ParkingBBoxResult struct {
	TotalLots       int64                 `json:"total_lots"`
	WithRealtime    int64                 `json:"with_realtime"`
	OccupiedRateAvg float64               `json:"occupied_rate_avg"`
	ByCity          []ParkingBBoxCityStat `json:"by_city"`
}

// GetParkingByBBox queries parking_realtime (Taipei) + parking_realtime_ntpc
// (NewTaipei) for lots whose point geometry falls inside the given lng/lat
// bbox. occupied_rate < 0 (the upstream "no realtime data" sentinel) is
// treated as 1.0 (fully occupied) per agent contract.
func GetParkingByBBox(lngMin, latMin, lngMax, latMax float64) (ParkingBBoxResult, error) {
	var res ParkingBBoxResult

	if !(lngMin < lngMax) || !(latMin < latMax) {
		return res, errors.New("bbox must satisfy lng_min < lng_max and lat_min < lat_max")
	}

	type row struct {
		Lots         int64
		WithRealtime int64
		OccSum       float64
	}

	queryFor := func(table string) (row, error) {
		var r row
		// `wkb_geometry && ST_MakeEnvelope(...)` uses the GiST index for fast
		// bbox prefilter; ST_Within would require an explicit polygon test
		// that adds nothing for points strictly inside the envelope.
		err := DBDashboard.Raw(`
			SELECT
				count(*)                                                       AS lots,
				count(*) FILTER (WHERE occupied_rate >= 0)                     AS with_realtime,
				COALESCE(sum(CASE WHEN occupied_rate < 0 THEN 1.0
				                  ELSE occupied_rate END), 0)                  AS occ_sum
			FROM `+table+`
			WHERE wkb_geometry && ST_MakeEnvelope(?, ?, ?, ?, 4326)
		`, lngMin, latMin, lngMax, latMax).Scan(&r).Error
		return r, err
	}

	tpe, err := queryFor("public.parking_realtime")
	if err != nil {
		return res, err
	}
	ntpc, err := queryFor("public.parking_realtime_ntpc")
	if err != nil {
		return res, err
	}

	avg := func(sum float64, n int64) float64 {
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}

	res.ByCity = []ParkingBBoxCityStat{
		{City: "taipei", Lots: tpe.Lots, WithRealtime: tpe.WithRealtime, OccupiedRateAvg: avg(tpe.OccSum, tpe.Lots)},
		{City: "newtaipei", Lots: ntpc.Lots, WithRealtime: ntpc.WithRealtime, OccupiedRateAvg: avg(ntpc.OccSum, ntpc.Lots)},
	}
	res.TotalLots = tpe.Lots + ntpc.Lots
	res.WithRealtime = tpe.WithRealtime + ntpc.WithRealtime
	res.OccupiedRateAvg = avg(tpe.OccSum+ntpc.OccSum, res.TotalLots)

	return res, nil
}
