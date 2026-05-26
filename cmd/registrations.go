package cmd

import "github.com/spf13/cobra"

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(_ *cobra.Command)          {}
func registerAccount(_ *cobra.Command)       {}
func registerDocs(_ *cobra.Command)          {}
