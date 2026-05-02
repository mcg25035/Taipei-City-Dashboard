// Package models — bbox-filtered YouBike station list consumed by /api/v1/agent.
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// YoubikeStation is one station inside the bbox plus its latest realtime
// availability. Returned by GetYoubikeStationsByBBox.
type YoubikeStation struct {
	StationID            string  `json:"station_id"`
	Name                 string  `json:"name"`
	Area                 string  `json:"area"`
	Addr                 string  `json:"addr"`
	Lng                  float64 `json:"lng"`
	Lat                  float64 `json:"lat"`
	County               string  `json:"county"` // "Taipei" | "New Taipei"
	Capacity             int     `json:"capacity"`
	AvailableRentBikes   int     `json:"available_rent_bikes"`
	AvailableReturnBikes int     `json:"available_return_bikes"`
	UpdatedAt            string  `json:"updated_at"` // upstream `mday` / `srcUpdateTime`
}

// Public realtime endpoints (no auth). Project DB has only realtime usage rows
// without coords (`tran_ubike_realtime{,_new_tpe}`) and no station-master
// table, so we pull from city OpenData. Cached for 60s — same cadence as the
// upstream "infoTime".
const (
	youbikeTPEURL  = "https://tcgbusfs.blob.core.windows.net/dotapp/youbike/v2/youbike_immediate.json"
	youbikeNTPCURL = "https://data.ntpc.gov.tw/api/datasets/010e5b15-3823-4b20-b401-b1cf000550c5/json"
	youbikeCacheTTL = 60 * time.Second
)

var (
	youbikeMu       sync.Mutex
	youbikeCache    []YoubikeStation
	youbikeCacheExp time.Time
	youbikeHTTP     = &http.Client{Timeout: 10 * time.Second}
)

// GetYoubikeStationsByBBox returns every YouBike station whose location falls
// inside the lng/lat bbox. Pulls Taipei + New Taipei from public OpenData
// endpoints with a 60s in-process cache.
func GetYoubikeStationsByBBox(lngMin, latMin, lngMax, latMax float64) ([]YoubikeStation, error) {
	if !(lngMin < lngMax) || !(latMin < latMax) {
		return nil, errors.New("bbox must satisfy lng_min < lng_max and lat_min < lat_max")
	}

	all, err := loadYoubikeStations()
	if err != nil {
		return nil, err
	}

	out := make([]YoubikeStation, 0, 64)
	for _, s := range all {
		if s.Lng >= lngMin && s.Lng <= lngMax && s.Lat >= latMin && s.Lat <= latMax {
			out = append(out, s)
		}
	}
	return out, nil
}

func loadYoubikeStations() ([]YoubikeStation, error) {
	youbikeMu.Lock()
	defer youbikeMu.Unlock()

	if time.Now().Before(youbikeCacheExp) && youbikeCache != nil {
		return youbikeCache, nil
	}

	type result struct {
		stations []YoubikeStation
		err      error
	}
	tpeCh := make(chan result, 1)
	ntpcCh := make(chan result, 1)

	go func() {
		s, err := fetchYoubikeTPE()
		tpeCh <- result{s, err}
	}()
	go func() {
		s, err := fetchYoubikeNTPC()
		ntpcCh <- result{s, err}
	}()

	tpe := <-tpeCh
	ntpc := <-ntpcCh

	// Tolerate a single-source failure: return whatever we got rather than
	// erroring out the whole call.
	if tpe.err != nil && ntpc.err != nil {
		return nil, fmt.Errorf("both upstreams failed: tpe=%v; ntpc=%v", tpe.err, ntpc.err)
	}

	merged := make([]YoubikeStation, 0, len(tpe.stations)+len(ntpc.stations))
	merged = append(merged, tpe.stations...)
	merged = append(merged, ntpc.stations...)

	youbikeCache = merged
	youbikeCacheExp = time.Now().Add(youbikeCacheTTL)
	return merged, nil
}

func fetchYoubikeTPE() ([]YoubikeStation, error) {
	body, err := httpGet(youbikeTPEURL)
	if err != nil {
		return nil, err
	}
	type row struct {
		Sno                  string  `json:"sno"`
		Sna                  string  `json:"sna"`
		Sarea                string  `json:"sarea"`
		Ar                   string  `json:"ar"`
		Mday                 string  `json:"mday"`
		Quantity             int     `json:"Quantity"`
		AvailableRentBikes   int     `json:"available_rent_bikes"`
		AvailableReturnBikes int     `json:"available_return_bikes"`
		Latitude             float64 `json:"latitude"`
		Longitude            float64 `json:"longitude"`
	}
	var rows []row
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode TPE youbike: %w", err)
	}
	out := make([]YoubikeStation, 0, len(rows))
	for _, r := range rows {
		out = append(out, YoubikeStation{
			StationID:            r.Sno,
			Name:                 r.Sna,
			Area:                 r.Sarea,
			Addr:                 r.Ar,
			Lng:                  r.Longitude,
			Lat:                  r.Latitude,
			County:               "Taipei",
			Capacity:             r.Quantity,
			AvailableRentBikes:   r.AvailableRentBikes,
			AvailableReturnBikes: r.AvailableReturnBikes,
			UpdatedAt:            r.Mday,
		})
	}
	return out, nil
}

func fetchYoubikeNTPC() ([]YoubikeStation, error) {
	// NTPC OpenData endpoint paginates at 1000 rows max.
	type row struct {
		Sno         string `json:"sno"`
		Sna         string `json:"sna"`
		Sarea       string `json:"sarea"`
		Ar          string `json:"ar"`
		Mday        string `json:"mday"`
		TotQuantity string `json:"tot_quantity"`
		SbiQuantity string `json:"sbi_quantity"`
		Bemp        string `json:"bemp"`
		Lat         string `json:"lat"`
		Lng         string `json:"lng"`
	}
	all := []row{}
	for page := 0; page < 10; page++ {
		url := fmt.Sprintf("%s?page=%d&size=1000", youbikeNTPCURL, page)
		body, err := httpGet(url)
		if err != nil {
			return nil, err
		}
		var rows []row
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("decode NTPC youbike page %d: %w", page, err)
		}
		if len(rows) == 0 {
			break
		}
		all = append(all, rows...)
		if len(rows) < 1000 {
			break
		}
	}

	out := make([]YoubikeStation, 0, len(all))
	for _, r := range all {
		lat, _ := strconv.ParseFloat(r.Lat, 64)
		lng, _ := strconv.ParseFloat(r.Lng, 64)
		cap, _ := strconv.Atoi(r.TotQuantity)
		rent, _ := strconv.Atoi(r.SbiQuantity)
		ret, _ := strconv.Atoi(r.Bemp)
		out = append(out, YoubikeStation{
			StationID:            r.Sno,
			Name:                 r.Sna,
			Area:                 r.Sarea,
			Addr:                 r.Ar,
			Lng:                  lng,
			Lat:                  lat,
			County:               "New Taipei",
			Capacity:             cap,
			AvailableRentBikes:   rent,
			AvailableReturnBikes: ret,
			UpdatedAt:            r.Mday,
		})
	}
	return out, nil
}

func httpGet(url string) ([]byte, error) {
	resp, err := youbikeHTTP.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
