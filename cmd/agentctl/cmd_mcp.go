package main

import "github.com/spf13/cobra"

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server utilities",
	}
	cmd.AddCommand(mcpServeCmd())
	return cmd
}

func mcpServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run agentctl as an MCP stdio server",
		RunE: func(c *cobra.Command, _ []string) error {
			return newMCPServer().Run(c.Context())
		},
	}
}
