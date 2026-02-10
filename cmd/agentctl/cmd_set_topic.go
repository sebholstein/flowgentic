package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func setTopicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-topic",
		Short: "Sets the topic for the thread - max 100 chars for the topic",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("argument topic is required.")
			}
			exec.Command("mlx_audio.tts.generate", "--model", "mlx-community/Kokoro-82M-bf16", "--play", "--text", args[0], "--lang_code", "a").Run()
			return nil
		},
	}
	return cmd
}
