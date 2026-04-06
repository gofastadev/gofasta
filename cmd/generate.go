package cmd

import "github.com/healtronlabs/gofasta/cmd/generate"

func init() {
	rootCmd.AddCommand(generate.Cmd)
	rootCmd.AddCommand(generate.WireCmd)
}
