package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Ackar/salta/geocoding"
	log "github.com/sirupsen/logrus"
)

type endpoint struct {
	geocoder *geocoding.ReverseGeocoder
}

func newEndpoint(geocoder *geocoding.ReverseGeocoder) *endpoint {
	return &endpoint{
		geocoder: geocoder,
	}
}

func (e *endpoint) LocationFromLatLong(w http.ResponseWriter, r *http.Request) {
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		http.Error(w, "invalid latitude", http.StatusBadRequest)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		http.Error(w, "invalid longitude", http.StatusBadRequest)
		return
	}

	res := e.geocoder.LocationFromLatLng(lat, lng)

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.WithError(err).Error("error encoding response")
	}
}
