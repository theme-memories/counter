package main

import (
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

type RequestParams struct {
	Scale   float64 `form:"scale"`
	Padding uint    `form:"padding"`
	Theme   string  `form:"theme"`
}

var (
	nameRegex = regexp.MustCompile(`^[a-z0-9]+$`)
)

func main() {
	mongodbURI := os.Getenv("MONGODB_URI")
	cfBeaconToken := os.Getenv("CF_BEACON_TOKEN")

	if mongodbURI == "" {
		log.Fatal("MONGODB_URI environment variable is required")
	}

	db := NewCounterDB(mongodbURI)
	db.StartSyncTicker(60 * time.Second)

	tm := NewThemeManager("assets/themes")

	r := gin.Default()

	r.LoadHTMLFiles("assets/index.html")
	r.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=600")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"CFToken": cfBeaconToken,
		})
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=2592000")
		c.File("assets/favicon.ico")
	})

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "alive")
	})

	r.GET("/:name", func(c *gin.Context) {
		name := c.Param("name")

		if !nameRegex.MatchString(name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid name: only a-z and 0-9 allowed"})
			return
		}

		var params RequestParams

		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
			return
		}
		if params.Scale == 0 {
			params.Scale = 1.0
		}
		if params.Padding == 0 {
			params.Padding = 8
		}
		if params.Theme == "" {
			params.Theme = "moebooru"
		}
		if params.Scale < 0.1 || params.Scale > 2.0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Scale must be between 0.1 and 2.0"})
			return
		}
		if params.Padding < 1 || params.Padding > 16 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Padding must be between 1 and 16"})
			return
		}
		if _, ok := tm.Themes[params.Theme]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme"})
			return
		}

		params.Scale = math.Round(params.Scale*100) / 100
		count := db.GetAndIncrement(name)
		svg := tm.GenerateSVG(count, params.Theme, params.Scale, int(params.Padding))

		c.Header("Content-Type", "image/svg+xml")
		c.Header("Cache-Control", "public, max-age=600")
		c.String(http.StatusOK, svg)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Printf("Server starting on :%s", port)
	r.Run(":" + port)
}
