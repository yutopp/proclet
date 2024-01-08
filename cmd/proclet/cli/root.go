package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var profilePath string

var rootCmd = &cobra.Command{
	Use: "proclet",
}

func init() {
	rootCmd.Flags().StringVar(&profilePath, "profilePath", "/tmp/proclet.profile.json", "profile path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
