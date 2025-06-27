package main

import (
	"github.com/axellelanca/urlshortener/cmd"
	_ "github.com/axellelanca/urlshortener/cmd/cli"    // Importe le package 'cli' pour que ses init() soient exécutés
	_ "github.com/axellelanca/urlshortener/cmd/server" // Importe le package 'server' pour que ses init() soient exécutés
	"log"
)

func main() {
	// TODO Exécute la commande racine de Cobra.
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatalf("Erreur lors de l'exécution de la commande : %v", err)
	}
}
