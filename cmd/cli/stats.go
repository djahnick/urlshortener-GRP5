package cli

import (
	"fmt"
	"log"
	"os"

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var shortCodeFlag string

var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Affiche les statistiques (nombre de clics) pour un lien court.",
	Long: `Cette commande permet de récupérer et d'afficher le nombre total de clics
pour une URL courte spécifique en utilisant son code.

Exemple:
  url-shortener stats --code="xyz123"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Récupérer la valeur du flag depuis cobra
		codeFlag, _ := cmd.Flags().GetString("code")

		if codeFlag == "" {
			fmt.Println("Erreur : le flag --code est requis.")
			os.Exit(1)
		}

		cfg := cmd2.Cfg

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Erreur ouverture DB : %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}
		defer sqlDB.Close()

		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)
		linkService := services.NewLinkService(linkRepo, clickRepo)

		link, totalClicks, err := linkService.GetLinkStats(codeFlag)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				fmt.Println("Erreur : code inconnu.")
			} else {
				fmt.Println("Erreur récupération stats :", err)
			}
			os.Exit(1)
		}

		if link == nil {
			fmt.Println("Erreur : le lien est introuvable.")
			os.Exit(1)
		}

		fmt.Printf("Statistiques pour le code court: %s\n", link.Shortcode)
		fmt.Printf("URL longue: %s\n", link.LongURL)
		fmt.Printf("Total de clics: %d\n", totalClicks)
	},
}

// Une seule fonction init()
func init() {
	StatsCmd.Flags().StringVar(&shortCodeFlag, "code", "", "Code du lien court")
	StatsCmd.MarkFlagRequired("code")
	cmd2.RootCmd.AddCommand(StatsCmd)
}
