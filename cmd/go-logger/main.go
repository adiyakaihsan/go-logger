package main

import (
	"log"
	"os"

	"github.com/adiyakaihsan/go-logger/pkg/app"
	"github.com/spf13/cobra"
)

var (
	serverCount int
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

	runCmd.Flags().IntVarP(&serverCount, "count", "c", 1, "Number of servers to run")
	runCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Init Port number")
	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}
