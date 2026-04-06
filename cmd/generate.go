package cmd

import "github.com/gofastadev/gofasta/cmd/generate"

func init() {
	rootCmd.AddCommand(generate.Cmd)
	rootCmd.AddCommand(generate.WireCmd)
}
