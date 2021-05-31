package main

import (
	_ "embed"

	"github.com/Ackar/salta/geocoding"
)

//go:embed schema.graphql
var schema string

type graphqlResolver struct {
	geocoder *geocoding.ReverseGeocoder
}

func newGraphqlResolver(geocoder *geocoding.ReverseGeocoder) *graphqlResolver {
	return &graphqlResolver{
		geocoder: geocoder,
	}
}

type location struct {
	Campus        *string
	Locality      *string
	MarketArea    *string
	Neighbourhood *string
	Borough       *string
	Microhood     *string
	County        *string
	MacroCounty   *string
	LocalAdmin    *string
	Region        *string
	MacroRegion   *string
	Country       *string
}

type locationFronLatLngInput struct {
	Latitude  float64
	Longitude float64
}

func (r *graphqlResolver) LocationFromLatLng(args struct {
	Input locationFronLatLngInput
}) *location {
	loc := r.geocoder.LocationFromLatLng(args.Input.Latitude, args.Input.Longitude)
	if loc == nil {
		return nil
	}

	var res location
	if loc.Campus != "" {
		res.Campus = &loc.Campus
	}
	if loc.Locality != "" {
		res.Locality = &loc.Locality
	}
	if loc.MarketArea != "" {
		res.MarketArea = &loc.MarketArea
	}
	if loc.Neighbourhood != "" {
		res.Neighbourhood = &loc.Neighbourhood
	}
	if loc.Borough != "" {
		res.Borough = &loc.Borough
	}
	if loc.Microhood != "" {
		res.Microhood = &loc.Microhood
	}
	if loc.County != "" {
		res.County = &loc.County
	}
	if loc.MacroCounty != "" {
		res.MacroCounty = &loc.MacroCounty
	}
	if loc.LocalAdmin != "" {
		res.LocalAdmin = &loc.LocalAdmin
	}
	if loc.Region != "" {
		res.Region = &loc.Region
	}
	if loc.MacroRegion != "" {
		res.MacroRegion = &loc.MacroRegion
	}
	if loc.Country != "" {
		res.Country = &loc.Country
	}

	return &res
}
