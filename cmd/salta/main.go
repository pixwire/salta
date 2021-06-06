package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Ackar/salta/geocoding"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("no config file provided")
	}

	viper.SetDefault("port", 8080)
	viper.SetDefault("repos.folder", "repos")
	viper.SetDefault("cache.folder", "cache")
	viper.SetDefault("enabled_place_types", []string{
		"locality",
		"neighbourhood",
		"borough",
		"microhood",
		"county",
		"macrocounty",
		"localadmin",
		"region",
		"macroregion",
		"country",
		"campus",
		"marketarea",
	})

	viper.SetConfigFile(os.Args[1])
	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Fatal("error reading config")
	}

	countries := viper.GetStringSlice("countries")
	enabledPlaceTypes := viper.GetStringSlice("enabled_place_types")
	reposFolder := viper.GetString("repos.folder")
	cacheFolder := viper.GetString("cache.folder")
	port := viper.GetInt("port")

	g := geocoding.NewReverseGeocoder(reposFolder, cacheFolder, countries, enabledPlaceTypes)
	err := g.Init()
	if err != nil {
		log.WithError(err).Fatal("error initializing geocoder")
	}

	r := newGraphqlResolver(g)
	schema := graphql.MustParseSchema(schema, r, graphql.UseFieldResolvers())

	ep := newEndpoint(g)

	http.HandleFunc("/location", ep.LocationFromLatLong)
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.WithField("port", port).Info("listening...")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
