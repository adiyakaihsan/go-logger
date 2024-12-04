package main

import (
	"log"
	"os"

	"github.com/adiyakaihsan/go-logger/pkg/app"
	"github.com/spf13/cobra"
)

var (
	indexName string
	serverPort  int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-logger",
		Short: "A logging server application",
		Long:  `A logging server application that can run multiple instances`,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the logging server(s)",
		Run: app.Run,
	}

	runCmd.Flags().StringVarP(&indexName, "index", "i", "index-storage/index", "Index Prefix")
	runCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Init Port number")
	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}
