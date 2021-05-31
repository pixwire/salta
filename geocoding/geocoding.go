package geocoding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/golang/geo/s2"
	geosimplification "github.com/hcliff/geo-simplification"
	geojson "github.com/paulmach/go.geojson"
	log "github.com/sirupsen/logrus"
)

// Location contains all the location information for a given location.
type Location struct {
	Campus        string `json:",omitempty"`
	Locality      string `json:",omitempty"`
	MarketArea    string `json:",omitempty"`
	Neighbourhood string `json:",omitempty"`
	Borough       string `json:",omitempty"`
	Microhood     string `json:",omitempty"`
	County        string `json:",omitempty"`
	MacroCounty   string `json:",omitempty"`
	LocalAdmin    string `json:",omitempty"`
	Region        string `json:",omitempty"`
	MacroRegion   string `json:",omitempty"`
	Country       string `json:",omitempty"`
}

func (l *Location) String() string {
	var s []string

	if l.Campus != "" {
		s = append(s, fmt.Sprintf("Campus:%s", l.Campus))
	}
	if l.Locality != "" {
		s = append(s, fmt.Sprintf("Locality:%s", l.Locality))
	}
	if l.MarketArea != "" {
		s = append(s, fmt.Sprintf("MarketArea:%s", l.MarketArea))
	}
	if l.Locality != "" {
		s = append(s, fmt.Sprintf("Locality:%s", l.Locality))
	}
	if l.Neighbourhood != "" {
		s = append(s, fmt.Sprintf("Neighbourhood:%s", l.Neighbourhood))
	}
	if l.Borough != "" {
		s = append(s, fmt.Sprintf("Borough:%s", l.Borough))
	}
	if l.Microhood != "" {
		s = append(s, fmt.Sprintf("Microhood:%s", l.Microhood))
	}
	if l.County != "" {
		s = append(s, fmt.Sprintf("County:%s", l.County))
	}
	if l.MacroCounty != "" {
		s = append(s, fmt.Sprintf("MacroCountry:%s", l.MacroCounty))
	}
	if l.LocalAdmin != "" {
		s = append(s, fmt.Sprintf("LocalAdmin:%s", l.LocalAdmin))
	}
	if l.Region != "" {
		s = append(s, fmt.Sprintf("Region:%s", l.Region))
	}
	if l.MacroRegion != "" {
		s = append(s, fmt.Sprintf("MacroRegion:%s", l.MacroRegion))
	}
	if l.Country != "" {
		s = append(s, fmt.Sprintf("Country:%s", l.Country))
	}

	return strings.Join(s, " ")
}

// ReverseGeocoder is a reverse geocoder.
type ReverseGeocoder struct {
	index             *s2.ShapeIndex
	reposFolder       string
	cacheFolder       string
	countries         []string
	enabledPlaceTypes []string
}

// NewReverseGeocoder returns a new geocoder from the given folders, countries and
// place types. reposFolder is the path to where WOF repos must be cloned,
// cacheFolder contains the cached version of the processed WOF geojsons.
func NewReverseGeocoder(reposFolder, cacheFolder string, countries, enabledPlaceTypes []string) *ReverseGeocoder {
	return &ReverseGeocoder{
		reposFolder:       reposFolder,
		cacheFolder:       cacheFolder,
		countries:         countries,
		enabledPlaceTypes: enabledPlaceTypes,

		index: s2.NewShapeIndex(),
	}
}

// LocationFromLatLng returns a Location from the given latitude and longitude.
func (g *ReverseGeocoder) LocationFromLatLng(lat, lng float64) *Location {
	q := s2.NewContainsPointQuery(g.index, s2.VertexModelOpen)
	shapes := q.ContainingShapes(s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng)))

	var res Location
	for _, r := range shapes {
		p := r.(*placePolygon)
		switch p.Place.PlaceType {
		case "locality":
			res.Locality = p.Place.Name
		case "neighbourhood":
			res.Neighbourhood = p.Place.Name
		case "borough":
			res.Borough = p.Place.Name
		case "microhood":
			res.Microhood = p.Place.Name
		case "county":
			res.County = p.Place.Name
		case "macrocounty":
			res.MacroCounty = p.Place.Name
		case "localadmin":
			res.LocalAdmin = p.Place.Name
		case "region":
			res.Region = p.Place.Name
		case "macroregion":
			res.MacroRegion = p.Place.Name
		case "country":
			res.Country = p.Place.Name
		case "campus":
			res.Campus = p.Place.Name
		case "marketarea":
			res.MarketArea = p.Place.Name
		default:
			log.Infof("unknown type %q", p.Place.PlaceType)
		}
	}
	return &res
}

// Init loads the data into the index.
// It first clones and updates the countries repositories, and the process all
// available geojson, using the cache when available.
func (g *ReverseGeocoder) Init() error {
	for _, c := range g.countries {
		err := g.loadCountry(c)
		if err != nil {
			return fmt.Errorf("error loading country %q: %w", c, err)
		}
	}
	return nil
}

func (g *ReverseGeocoder) loadCountry(country string) error {
	err := g.cloneAndUpdateRepository(country)
	if err != nil {
		return fmt.Errorf("error updating repository: %w", err)
	}

	err = g.indexCountry(country)
	if err != nil {
		return fmt.Errorf("error indexing country: %w", err)
	}

	return nil
}

func (g *ReverseGeocoder) cloneAndUpdateRepository(country string) error {
	if err := g.createReposFolder(); err != nil {
		return fmt.Errorf("could not create repos folder: %w", err)
	}

	repoPath := g.repoPath(country)

	info, err := os.Stat(repoPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking %q: %w", repoPath, err)
	}

	if info != nil && !info.IsDir() {
		return fmt.Errorf("%q is not a directory", repoPath)
	}

	// we can't use go-git because WOF uses git LFS
	gitURL := fmt.Sprintf("https://github.com/whosonfirst-data/whosonfirst-data-admin-%s.git", country)
	log.WithField("repository", gitURL).Info("cloning repository")
	if os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", gitURL, repoPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error cloning repository: %w", err)
		}
	}

	log.WithField("country", country).Info("updating repository")
	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("error updating repository: %w", err)
	}

	return nil
}

var crcTable = crc64.MakeTable(crc64.ISO)

// indexCountry iterates over all geojson files for a country and add them to the index.
// If an up-to-date cached version exists indexCountry loads it, otherwise it
// processes the source file and creates a cache file.
func (g *ReverseGeocoder) indexCountry(country string) error {
	repoPath := g.repoPath(country)

	log.WithField("country", country).Info("processing country files, this might take a while...")

	err := g.createCacheFolder(country)
	if err != nil {
		return fmt.Errorf("could not create cache directory: %w", err)
	}

	concurrent := runtime.GOMAXPROCS(0)
	filesChan := make(chan string, concurrent)
	polygonChan := make(chan *placePolygon, concurrent)

	// start geojson workers
	var filesWG sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		filesWG.Add(1)
		go func() {
			defer filesWG.Done()

			for path := range filesChan {
				cache, err := g.loadCachedPolygons(country, path)
				if err != nil {
					log.WithError(err).Error("error loading cached polygon: %w", err)
					continue
				}
				if cache != nil {
					if !cache.Valid {
						continue
					}
					for _, p := range cache.PlacePolygons() {
						polygonChan <- p
					}
					continue
				}

				polygons, err := g.processGeojson(country, path)
				if err != nil {
					log.WithError(err).Error("error processing geojson %q", path)
					continue
				}
				for _, p := range polygons {
					polygonChan <- p
				}
			}
		}()
	}

	// start indexer
	var polygonWG sync.WaitGroup
	polygonWG.Add(1)
	go func() {
		defer polygonWG.Done()

		var count int
		for p := range polygonChan {
			count++
			if count%1000 == 0 {
				log.WithField("country", country).Infof("loaded %d polygons", count)
			}

			if !g.placeTypeEnabled(p.Place.PlaceType) {
				continue
			}

			g.index.Add(p)
		}
	}()

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".geojson") {
			return nil
		}

		filesChan <- path

		return nil
	})
	close(filesChan)

	if err != nil {
		return fmt.Errorf("error processing country files: %w", err)
	}

	filesWG.Wait()
	close(polygonChan)

	polygonWG.Wait()

	return nil
}

func fileHash(path string) (string, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}
	return fmt.Sprintf("%x", crc64.Checksum(fileBytes, crcTable)), nil
}

func (g *ReverseGeocoder) loadCachedPolygons(country, path string) (*cachedFile, error) {
	cachePath := g.cacheFile(country, path)
	if _, err := os.Stat(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	b, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	var cache cachedFile
	err = json.Unmarshal(b, &cache)
	if err != nil {
		return nil, err
	}
	hash, err := fileHash(path)
	if err != nil {
		return nil, fmt.Errorf("could not get file hash: %w", err)
	}

	if hash != cache.Hash {
		// file has changed
		return nil, nil
	}

	return &cache, nil
}

// processGeojson reads the given geojson file and returns a list of simplified
// polygons.
func (g *ReverseGeocoder) processGeojson(country, path string) ([]*placePolygon, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	feature, err := geojson.UnmarshalFeature(b)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling geojson: %w", err)
	}

	name, ok := feature.Properties["wof:name"].(string)
	if !ok || name == "" {
		return nil, g.cacheInvalid(country, path)
	}
	placeType, ok := feature.Properties["wof:placetype"].(string)
	if !ok || placeType == "" {
		return nil, g.cacheInvalid(country, path)
	}

	if !feature.Geometry.IsPolygon() && !feature.Geometry.IsMultiPolygon() {
		return nil, g.cacheInvalid(country, path)
	}

	var srcPolygons [][][][]float64
	if feature.Geometry.IsMultiPolygon() {
		srcPolygons = feature.Geometry.MultiPolygon
	} else {
		srcPolygons = [][][][]float64{feature.Geometry.Polygon}
	}

	var res []*placePolygon
	var polygons []*s2.Polygon
	for _, p := range srcPolygons {
		s2p, err := convertToS2Polygon(p)
		if err != nil {
			log.WithError(err).Error("ignoring polygon")
			continue
		}
		if s2p != nil {
			polygons = append(polygons, s2p)
			res = append(res, &placePolygon{
				Polygon: s2p,
				Place: place{
					Name:      name,
					PlaceType: placeType,
				},
			})
		}
	}

	hash, err := fileHash(path)
	if err != nil {
		return nil, fmt.Errorf("could not get file hash: %w", err)
	}

	err = g.writeCache(country, path, &cachedFile{
		Hash:  hash,
		Valid: true,
		Place: place{
			Name:      name,
			PlaceType: placeType,
		},
		Polygons: polygons,
	})
	if err != nil {
		return nil, fmt.Errorf("error writing cache: %w", err)
	}

	return res, nil
}

func convertToS2Polygon(p [][][]float64) (*s2.Polygon, error) {
	loops := make([]*s2.Loop, 0, len(p))
	for _, x := range p {
		loop := toLoop(x)
		threshold := 0.0001
		minPointsToKeep := 0
		avoidIntersections := true
		loop, err := geosimplification.SimplifyLoop(loop, threshold, minPointsToKeep, avoidIntersections)
		if err != nil {
			return nil, fmt.Errorf("error simplifying loop: %w", err)
		}
		loops = append(loops, loop)
	}

	res := s2.PolygonFromLoops(loops)
	// if the area is huge it usually means the polygon is inverted, so try to
	// invert it and if it's still huge skip it...
	if res.RectBound().Area() > 10 {
		res.Invert()
		if res.RectBound().Area() > 10 {
			return nil, nil
		}
	}

	return res, nil
}

func (g *ReverseGeocoder) createReposFolder() error {
	err := os.MkdirAll(g.reposFolder, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (g *ReverseGeocoder) createCacheFolder(country string) error {
	err := os.MkdirAll(fmt.Sprintf("%s/%s", g.cacheFolder, country), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (g *ReverseGeocoder) cacheInvalid(country, path string) error {
	hash, err := fileHash(path)
	if err != nil {
		return fmt.Errorf("could not get file hash: %w", err)
	}

	return g.writeCache(country, path, &cachedFile{
		Hash:  hash,
		Valid: false,
	})
}

func (g *ReverseGeocoder) writeCache(country, path string, cache *cachedFile) error {
	f, err := os.Create(g.cacheFile(country, path))
	if err != nil {
		return fmt.Errorf("unable to create cache file: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(cache)
}

func (g *ReverseGeocoder) repoPath(country string) string {
	return fmt.Sprintf("%s/%s", g.reposFolder, country)
}

func (g *ReverseGeocoder) cacheFile(country, path string) string {
	return fmt.Sprintf("%s/%s/%s", g.cacheFolder, country, filepath.Base(path))
}

func (g *ReverseGeocoder) placeTypeEnabled(placetype string) bool {
	if len(g.enabledPlaceTypes) == 0 {
		// all enabled
		return true
	}

	for _, a := range g.enabledPlaceTypes {
		if placetype == a {
			return true
		}
	}
	return false
}

func rectToPolygon(r s2.Rect) *s2.Polygon {
	var pts []s2.Point

	pts = append(pts, s2.PointFromLatLng(s2.LatLng{Lat: r.Lo().Lat, Lng: r.Lo().Lng}))
	pts = append(pts, s2.PointFromLatLng(s2.LatLng{Lat: r.Lo().Lat, Lng: r.Hi().Lng}))
	pts = append(pts, s2.PointFromLatLng(s2.LatLng{Lat: r.Hi().Lat, Lng: r.Hi().Lng}))
	pts = append(pts, s2.PointFromLatLng(s2.LatLng{Lat: r.Hi().Lat, Lng: r.Lo().Lng}))

	loop := s2.LoopFromPoints(pts)

	return s2.PolygonFromLoops([]*s2.Loop{loop})
}

func toLoop(points [][]float64) *s2.Loop {
	var pts []s2.Point
	ptsMap := make(map[s2.Point]struct{}, len(points))
	for _, pt := range points {
		p := s2.PointFromLatLng(s2.LatLngFromDegrees(pt[1], pt[0]))
		if _, ok := ptsMap[p]; ok {
			continue
		}
		ptsMap[p] = struct{}{}
		pts = append(pts, p)
	}
	return s2.LoopFromPoints(pts)
}

type place struct {
	Name      string
	PlaceType string
}

type placePolygon struct {
	*s2.Polygon
	Place place
}

type polygons []*s2.Polygon

func (p polygons) MarshalJSON() ([]byte, error) {
	encoded := make([][]byte, 0, len(p))
	for _, e := range p {
		var buf bytes.Buffer
		err := e.Encode(&buf)
		if err != nil {
			return nil, fmt.Errorf("error encoding polygon: %w", err)
		}
		encoded = append(encoded, buf.Bytes())
	}
	return json.Marshal(encoded)
}

func (p *polygons) UnmarshalJSON(data []byte) error {
	var encoded [][]byte
	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return err
	}

	for _, e := range encoded {
		var s2Polygon s2.Polygon
		err := s2Polygon.Decode(bytes.NewReader(e))
		if err != nil {
			return fmt.Errorf("error decoding polygon: %w", err)
		}
		*p = append(*p, &s2Polygon)
	}

	return nil
}

type cachedFile struct {
	Hash     string
	Valid    bool
	Place    place
	Polygons polygons
}

func (c *cachedFile) PlacePolygons() []*placePolygon {
	res := make([]*placePolygon, 0, len(c.Polygons))
	for _, p := range c.Polygons {
		res = append(res, &placePolygon{
			Polygon: p,
			Place:   c.Place,
		})
	}

	return res
}
