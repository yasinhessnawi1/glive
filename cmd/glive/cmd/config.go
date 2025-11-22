package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/glive/core/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GLive configuration",
	Long:  `Configure GLive settings like API keys, workspace directory, and default mode.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api-key          - DeepSeek API key
  api-provider     - AI provider (deepseek, openai, claude)
  api-endpoint     - Custom API endpoint
  default-mode     - Default execution mode (auto, assisted, manual)
  workspace-dir    - Directory for cloned projects
  max-concurrent   - Maximum concurrent projects
  enable-sandbox   - Enable sandbox mode (true/false)
  agent-port       - Port for local agent`,
	Args: cobra.ExactArgs(2),
	Run:  setConfig,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value(s)",
	Args:  cobra.MaximumNArgs(1),
	Run:   getConfig,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration",
	Run:   listConfig,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
}

func setConfig(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	cfgManager, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Handle special conversions
	var valueToSet interface{} = value

	switch key {
	case "max-concurrent", "agent-port":
		intVal, err := strconv.Atoi(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid integer value for %s: %v\n", key, err)
			os.Exit(1)
		}
		valueToSet = intVal
	case "enable-sandbox":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid boolean value for %s: %v\n", key, err)
			os.Exit(1)
		}
		valueToSet = boolVal
	}

	if err := cfgManager.Set(key, valueToSet); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to set %s: %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("✓ Set %s = %v\n", key, value)
}

func getConfig(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		listConfig(cmd, args)
		return
	}

	key := args[0]

	cfgManager, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	switch key {
	case "api-key":
		if cfg.APIKey != "" {
			fmt.Printf("api-key: %s\n", cfg.APIKey)
		} else {
			fmt.Println("api-key: (not set)")
		}
	case "api-provider":
		fmt.Printf("api-provider: %s\n", cfg.APIProvider)
	case "api-endpoint":
		if cfg.APIEndpoint != "" {
			fmt.Printf("api-endpoint: %s\n", cfg.APIEndpoint)
		} else {
			fmt.Println("api-endpoint: (not set)")
		}
	case "default-mode":
		fmt.Printf("default-mode: %s\n", cfg.DefaultMode)
	case "workspace-dir":
		fmt.Printf("workspace-dir: %s\n", cfg.WorkspaceDir)
	case "max-concurrent":
		fmt.Printf("max-concurrent: %d\n", cfg.MaxConcurrent)
	case "enable-sandbox":
		fmt.Printf("enable-sandbox: %t\n", cfg.EnableSandbox)
	case "agent-port":
		fmt.Printf("agent-port: %d\n", cfg.AgentPort)
	default:
		fmt.Fprintf(os.Stderr, "❌ Unknown config key: %s\n", key)
		os.Exit(1)
	}
}

func listConfig(cmd *cobra.Command, args []string) {
	cfgManager, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	fmt.Println("Current configuration:")

	if cfg.APIKey != "" {
		// Mask the API key for security
		maskedKey := cfg.APIKey
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:4] + "..." + maskedKey[len(maskedKey)-4:]
		}
		fmt.Printf("  api-key:        %s\n", maskedKey)
	} else {
		fmt.Println("  api-key:        (not set)")
	}

	fmt.Printf("  api-provider:   %s\n", cfg.APIProvider)

	if cfg.APIEndpoint != "" {
		fmt.Printf("  api-endpoint:   %s\n", cfg.APIEndpoint)
	} else {
		fmt.Println("  api-endpoint:   (default)")
	}

	fmt.Printf("  default-mode:   %s\n", cfg.DefaultMode)
	fmt.Printf("  workspace-dir:  %s\n", cfg.WorkspaceDir)
	fmt.Printf("  max-concurrent: %d\n", cfg.MaxConcurrent)
	fmt.Printf("  enable-sandbox: %t\n", cfg.EnableSandbox)
	fmt.Printf("  agent-port:     %d\n", cfg.AgentPort)
}
