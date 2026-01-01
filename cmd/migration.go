/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/bootstrap"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/spf13/cobra"
)

var migrationCmd = &cobra.Command{
	Use:   "migration [command] [version]",
	Short: "Run database migrations",
	Long: `Commands:
  up           Apply all available migrations
  down         Roll back the last migration
  status       Show migration status
  version      Show current version
  redo         Roll back and reapply the last migration
  reset        Roll back all migrations
  up-to        Migrate up to a specific version
  down-to      Migrate down to a specific version`,
	Run: func(cmd *cobra.Command, args []string) {
		action := "up"
		if len(args) > 0 {
			action = args[0]
		}

		var version int64
		if action == "up-to" || action == "down-to" {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "version is required for", action)
				os.Exit(1)
			}
			v, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid version:", err)
				os.Exit(1)
			}
			version = v
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "config error:", err)
			os.Exit(1)
		}

		if err := bootstrap.Migrate(cmd.Context(), cfg, action, version); err != nil {
			fmt.Fprintln(os.Stderr, "migration error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(migrationCmd)
}
