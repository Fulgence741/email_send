package email

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var handlers = map[string]gin.HandlerFunc{
	//Toutes les ressources de l'API
	"emails":       AddEmailHandlers,
	"applications": AddApplicationhandlers,
	//"piece-jointe":      PieceJointehandlers,
	//"tentatives-envoie": TentativeEnvoiehandlers,
}

func Dispatch(c *gin.Context) {
	// Récupérer la ressource
	ressource := c.Param("ressource")
	ressource = strings.ToLower(ressource)
	ressource = strings.TrimSpace(ressource)

	//Recherche du handler

	handler, exists := handlers[ressource]

	// Vérfier que la ressource existe

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":  false,
			"error":    "RESOURCE_NOT_FOUD",
			"messaage": "Ressource non disponible",
		})

		return
	}

	// Exécuter le handler

	handler(c)
}
