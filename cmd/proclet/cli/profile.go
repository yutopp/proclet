package cli

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/yutopp/proclet/pkg/domain"
	"github.com/yutopp/proclet/pkg/server"
)

var profileCmd = &cobra.Command{
	Use: "profile",
	Run: func(cmd *cobra.Command, args []string) {
		profileRepo := server.NewProfileFromFile(profilePath)
		err := profileRepo.Save(&domain.Profile{
			Languages: L,
		})
		if err != nil {
			log.Panicf("failed to run: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
}

var L = []domain.Language{
	{
		ID:       "test-shell",
		ShowName: "Test Shell",

		Processors: []domain.Processor{
			{
				ID:       "alpine-sh-latest",
				ShowName: "sh (alpine:latest)",

				DockerImage: "alpine:latest",

				DefaultFilename: "main.sh",

				Tasks: []domain.Task{
					{
						ID:       "run",
						Kind:     "action",
						ShowName: "Run",

						Compile: nil,
						Run: &domain.PhasedTask{
							Cmd: []string{"sh", "main.sh"},
						},
					},
				},
			},
		},
	},
}
