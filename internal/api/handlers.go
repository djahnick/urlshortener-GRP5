package api

import (
	"errors"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"time"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Pour gérer gorm.ErrRecordNotFound
)

var ClickEventsChannel chan models.ClickEvent

// ClickEventsChannel est le channel global (ou injecté) utilisé pour envoyer les événements de clic
// aux workers asynchrones. Il est bufferisé pour ne pas bloquer les requêtes de redirection.

// SetupRoutes configure toutes les routes de l'API Gin et injecte les dépendances nécessaires
func SetupRoutes(router *gin.Engine, linkService *services.LinkService) {
	// Le channel est initialisé ici.
	if ClickEventsChannel == nil {
		// TODO Créer le channel ici (make), il doit être bufférisé
		// La taille du buffer doit être configurable via Viper (cfg.Analytics.BufferSize)
		ClickEventsChannel = make(chan models.ClickEvent, viper.GetInt("analytics.buffer_size"))

	}

	// TODO : Route de Health Check , /health
	//router.GET()
	router.GET("/health", HealthCheckHandler)
	router.POST("/api/v1/links", CreateShortLinkHandler(linkService))
	router.GET("/api/v1/links/:shortCode/stats", GetLinkStatsHandler(linkService))

	// Route de Redirection (au niveau racine pour les short codes)
	router.GET("/:shortCode", RedirectHandler(linkService))
	//Ajout des middlewares
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

}

// HealthCheckHandler gère la route /health pour vérifier l'état du service.
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest représente le corps de la requête JSON pour la création d'un lien.
type CreateLinkRequest struct {
	LongURL string `json:"long_url" binding:"required,url"` // 'binding:required' pour validation, 'url' pour format URL
}

// CreateShortLinkHandler gère la création d'une URL courte.
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest
		// TODO : Tente de lier le JSON de la requête à la structure CreateLinkRequest.
		// Gin gère la validation 'binding'.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		// TODO: Appeler le LinkService (CreateLink pour créer le nouveau lien.
		link, err := linkService.CreateLink(req.LongURL)
		if err != nil {
			log.Printf("Error creating short link: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
			return
		}

		baseURL := viper.GetString("server.base_url")
		// Retourne le code court et l'URL longue dans la réponse JSON.
		// TODO Choisir le bon code HTTP
		c.JSON(http.StatusCreated, gin.H{
			"short_code":     link.Shortcode,
			"long_url":       link.LongURL,
			"full_short_url": baseURL + "/" + link.Shortcode,
		})
	}
}

// RedirectHandler gère la redirection d'une URL courte vers l'URL longue et l'enregistrement asynchrone des clics.
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL avec c.Param
		shortCode := c.Param("shortCode")

		// TODO 2: Récupérer l'URL longue associée au shortCode depuis le linkService (GetLinkByShortCode)
		link, err := linkService.GetLinkByShortCode(shortCode)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short link not found"})
				return
			}
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// TODO 3: Créer un ClickEvent avec les informations pertinentes.
		clickEvent := models.ClickEvent{
			LinkID:    link.ID,
			Timestamp: time.Now(),
			UserAgent: c.Request.UserAgent(),
			IPAddress: c.Request.RemoteAddr,
		}

		// TODO 4: Envoyer le ClickEvent dans le ClickEventsChannel avec le Multiplexage.
		// Utilise un `select` avec un `default` pour éviter de bloquer si le channel est plein.
		// Pour le default, juste un message à afficher :
		// log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		select {
		case ClickEventsChannel <- clickEvent:
		default:
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}
		// TODO 5: Effectuer la redirection HTTP 302 (StatusFound) vers l'URL longue.
		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler gère la récupération des statistiques pour un lien spécifique.
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortCode := c.Param("shortCode")

		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short link not found"})
				return
			}
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.Shortcode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
