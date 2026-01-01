/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/bootstrap"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/spf13/cobra"
)

var seedCount int
var seedBatchSize int

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with sample data",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "config error:", err)
			os.Exit(1)
		}
		if err := bootstrap.Seed(cmd.Context(), cfg, seedCount, seedBatchSize); err != nil {
			fmt.Fprintln(os.Stderr, "seed error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	seedCmd.Flags().IntVar(&seedCount, "count", 10, "number of users to seed")
	seedCmd.Flags().IntVar(&seedBatchSize, "batch-size", 100, "batch size for inserts")
	rootCmd.AddCommand(seedCmd)
}
