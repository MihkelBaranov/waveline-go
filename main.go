package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mdp/qrterminal"
)

// Track Track structure
type Track struct {
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Track     int    `json:"track"`
	ID        string `json:"id"`
	Path      string `json:"path"`
	Favourite bool   `json:"favourite"`
	AlbumID   string `json:"albumId"`
	Size      int64  `json:"size"`
}

// Playlist Playlist structure
type Playlist struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Art    string  `json:"art"`
	Tracks []Track `json:"tracks"`
}

// Info Info structure
type Info struct {
	Start     string  `json:"start"`
	End       string  `json:"end"`
	Seconds   float64 `json:"seconds"`
	Tracks    int     `json:"tracks"`
	Playlists int     `json:"playlists"`
	Mount     string  `json:"mount"`
	Size      int64   `json:"size"`
}

// Config Config structure
type Config struct {
	Path string `json:"path"`
	Port string `json:"port"`
	Auth struct {
		Enabled  bool   `json:"enabled"`
		Password string `json:"password"`
		Username string `json:"username"`
	}
}

var (
	musicLibrary string
	port         string
	config       Config

	_storage     string
	_collections []string
	_data        interface{}
)

const (
	artPath = "./.art"
)

func main() {
	config = getConfig("./config.json")
	musicLibrary = config.Path
	port = config.Port
	open("./.cache")

	e := echo.New()
	e.HideBanner = true

	if config.Auth.Enabled {
		e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			if username == config.Auth.Username && password == config.Auth.Password {
				return true, nil
			}
			return false, nil
		}))
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/sync", sync)
	e.GET("/info", info)

	e.GET("/tracks", tracks)
	e.GET("/playlists", playlists)

	e.GET("/stream/:id", stream)
	e.GET("/favourite/:id", favourite)
	e.GET("/favourites", favourites)
	e.GET("/art/:id", art)

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("error: " + err.Error() + "\n")
		os.Exit(1)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				qrterminal.Generate("http://"+ipnet.IP.String()+":"+port, qrterminal.M, os.Stdout)
			}
		}
	}

	// Start server
	e.Logger.Fatal(e.Start(string(fmt.Sprintf(":%s", port))))

}

func info(c echo.Context) error {
	info := Info{}

	if err := read("metadata.json", &info); err != nil {
		return err
	}

	response := make(map[string]interface{})
	response["message"] = info
	return c.JSON(http.StatusOK, response)
}

func stream(c echo.Context) error {
	tracks := []Track{}
	if err := read("tracks.json", &tracks); err != nil {
		return err
	}

	track := Track{}

	for i := range tracks {
		if tracks[i].ID == c.Param("id") {
			track = tracks[i]
			break
		}
	}

	return c.File(track.Path)
}

func favourite(c echo.Context) error {
	tracks := []Track{}
	if err := read("tracks.json", &tracks); err != nil {
		return err
	}

	for i := range tracks {
		if tracks[i].ID == c.Param("id") {
			if tracks[i].Favourite {
				tracks[i].Favourite = false
			} else {
				tracks[i].Favourite = true
			}

			break

		}
	}

	write("tracks.json", tracks)

	response := make(map[string]interface{})
	response["message"] = true
	return c.JSON(http.StatusOK, response)
}

func favourites(c echo.Context) error {
	tracks := []Track{}
	favourites := []Track{}

	if err := read("tracks.json", &tracks); err != nil {
		return err
	}

	for i := range tracks {
		if tracks[i].Favourite {
			favourites = append(favourites, tracks[i])
		}

	}

	response := make(map[string]interface{})
	response["message"] = favourites
	return c.JSON(http.StatusOK, response)
}

func art(c echo.Context) error {
	file := artPath + "/" + c.Param("id")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		file = "./resources/placeholder.png"
	}
	return c.File(file)
}

func playlists(c echo.Context) error {
	playlists := []Playlist{}

	if err := read("playlists.json", &playlists); err != nil {
		fmt.Println("Error", err)
	}

	skip, err := strconv.Atoi(c.QueryParam("skip"))
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 20
	}
	response := make(map[string]interface{})
	response["message"] = playlistPagination(playlists, skip, limit)
	return c.JSON(http.StatusOK, response)
}

// Handler
func tracks(c echo.Context) error {
	tracks := []Track{}
	if err := read("tracks.json", &tracks); err != nil {
		fmt.Println("Error", err)
	}

	results := []Track{}

	search := strings.ToLower(c.QueryParam("search"))

	if len(search) > 0 {
		for i := range tracks {
			if strings.Contains(strings.ToLower(tracks[i].Title), search) {
				results = append(results, tracks[i])
			}

			if strings.Contains(strings.ToLower(tracks[i].Album), search) {
				results = append(results, tracks[i])
			}

			if strings.Contains(strings.ToLower(tracks[i].Artist), search) {
				results = append(results, tracks[i])
			}

		}

		response := make(map[string]interface{})
		response["message"] = results
		return c.JSON(http.StatusOK, response)
	}

	response := make(map[string]interface{})
	response["message"] = tracks
	return c.JSON(http.StatusOK, response)
}

// Handler
func sync(c echo.Context) error {
	root := musicLibrary
	results := _buildLibrary(root)
	response := make(map[string]interface{})
	response["message"] = results
	return c.JSON(http.StatusOK, response)
}

func _buildLibrary(root string) []Playlist {
	playlists := []Playlist{}
	metaInfo := Info{}

	tracks := getTracks(root)

	start := time.Now()

	metaInfo.Tracks = len(tracks)
	metaInfo.Mount = root

	metaInfo.Start = start.Format(time.RFC3339)
	write("tracks.json", tracks)

	for i := range tracks {
		index := index(playlists, tracks[i])

		metaInfo.Size += tracks[i].Size

		if index != -1 {
			playlists[index].Tracks = append(playlists[index].Tracks, tracks[i])
		} else {
			obj := Playlist{
				Title:  tracks[i].Album,
				ID:     tracks[i].AlbumID,
				Artist: tracks[i].Artist,
				Art:    tracks[i].ID,
			}
			obj.Tracks = append(obj.Tracks, tracks[i])

			playlists = append(playlists, obj)

		}
	}

	metaInfo.Playlists = len(playlists)

	write("playlists.json", playlists)

	t := time.Now()

	metaInfo.End = t.Format(time.RFC3339)
	metaInfo.Seconds = t.Sub(start).Seconds()

	write("metadata.json", metaInfo)

	return playlists
}

func index(playlists []Playlist, track Track) int {
	for i := range playlists {
		if playlists[i].ID == track.AlbumID {
			return i
		}
	}
	return -1
}

func getTracks(root string) []Track {
	tracks := []Track{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".flac") {
				f, err := os.Open(path)
				if err != nil {
					return nil
				}
				defer f.Close()

				size := info.Size()

				m, err := tag.ReadFrom(f)
				if err != nil {
					return nil
				}
				artist := strings.TrimSpace(m.Artist())
				title := strings.TrimSpace(m.Title())
				album := strings.TrimSpace(m.Album())
				trackNo, _ := m.Track()
				hash := md5Hash(title + album + artist)
				albumID := md5Hash(album + artist)

				if m.Picture() != nil {
					saveAlbumArt(m, hash)
				}

				result := Track{ID: hash, AlbumID: albumID, Size: size, Artist: artist, Title: title, Album: album, Track: trackNo, Path: path}

				tracks = append(tracks, result)
			}
		}
		return nil

	})
	return tracks
}

func md5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func saveAlbumArt(m tag.Metadata, hash string) error {
	os.Mkdir(artPath, 0775)
	art, err := os.OpenFile(fmt.Sprintf("%s/%s", artPath, hash), os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC, 0644)
	if err != nil {
		return nil
	}
	io.Copy(art, bytes.NewReader(m.Picture().Data))
	art.Close()
	return nil
}

func playlistPagination(x []Playlist, skip int, size int) []Playlist {
	if skip > len(x) {
		skip = len(x)
	}

	end := skip + size
	if end > len(x) {
		end = len(x)
	}

	return x[skip:end]
}

func open(path string) {
	os.Mkdir(path, 0775)
	_storage = path
}

func collections() []string {
	files, err := ioutil.ReadDir(_storage)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			_collections = append(_collections, f.Name())
		}
	}

	return _collections
}

func find(collection string, query interface{}, v interface{}) error {
	return nil
}

func update(collection string, data interface{}, v interface{}) error {
	return nil
}

func read(collection string, v interface{}) error {
	bytes, err := ioutil.ReadFile(_storage + "/" + collection)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, &v)
}

func write(collection string, d interface{}) error {
	jsonf, _ := json.Marshal(d)
	err := ioutil.WriteFile(_storage+"/"+collection, jsonf, 0644)

	if err != nil {
		return err
	}

	return nil
}

func getConfig(file string) Config {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		panic(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}
