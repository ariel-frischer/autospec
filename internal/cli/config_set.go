package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in user or project config.

By default, sets the value in the user-level config (~/.config/autospec/config.yml).
Use --project to set in the project-level config (.autospec/config.yml).

The value type is automatically inferred and validated against the expected type.`,
	Example: `  # Set max retries (user config)
  autospec config set max_retries 5

  # Enable notifications (user config)
  autospec config set notifications.enabled true

  # Set timeout at project level
  autospec config set timeout 3600 --project

  # Set notification type
  autospec config set notifications.type sound`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get the current value of a configuration key.

Shows the effective value and which config file it came from.`,
	Example: `  # Get max retries
  autospec config get max_retries

  # Get notification enabled status
  autospec config get notifications.enabled`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configToggleCmd = &cobra.Command{
	Use:   "toggle <key>",
	Short: "Toggle a boolean configuration value",
	Long: `Toggle a boolean configuration value between true and false.

If the key doesn't exist, it will be created as true.
Only works with boolean configuration keys.`,
	Example: `  # Toggle notifications enabled
  autospec config toggle notifications.enabled

  # Toggle skip preflight at project level
  autospec config toggle skip_preflight --project`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigToggle,
}

var configKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List all available configuration keys",
	Long:  `Display all valid configuration keys with their types and descriptions.`,
	RunE:  runConfigKeys,
}

func init() {
	// Add subcommands to config
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configToggleCmd)
	configCmd.AddCommand(configKeysCmd)

	// Add flags for set command
	configSetCmd.Flags().Bool("user", false, "Set in user-level config (default)")
	configSetCmd.Flags().Bool("project", false, "Set in project-level config")

	// Add flags for get command
	configGetCmd.Flags().Bool("user", false, "Get from user-level config only")
	configGetCmd.Flags().Bool("project", false, "Get from project-level config only")

	// Add flags for toggle command
	configToggleCmd.Flags().Bool("user", false, "Toggle in user-level config (default)")
	configToggleCmd.Flags().Bool("project", false, "Toggle in project-level config")
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	out := cmd.OutOrStdout()

	filePath, scope, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}

	if err := config.SetConfigValue(filePath, key, value); err != nil {
		return fmt.Errorf("setting config value: %w", err)
	}

	fmt.Fprintf(out, "Set %s = %s in %s config (%s)\n", key, value, scope, filePath)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	out := cmd.OutOrStdout()

	// Validate key exists
	if _, err := config.GetKeySchema(key); err != nil {
		return formatUnknownKeyError(key)
	}

	keyPath, err := config.ParseKeyPath(key)
	if err != nil {
		return fmt.Errorf("parsing key path: %w", err)
	}

	useUser, _ := cmd.Flags().GetBool("user")
	useProject, _ := cmd.Flags().GetBool("project")

	if useUser && useProject {
		return fmt.Errorf("--user and --project are mutually exclusive")
	}

	// If specific scope requested, check only that file
	if useUser || useProject {
		return getFromSpecificScope(out, key, keyPath, useProject)
	}

	// Show effective value (project overrides user)
	return getEffectiveValue(out, key, keyPath)
}

func getFromSpecificScope(out io.Writer, key string, keyPath []string, useProject bool) error {
	var filePath string
	var scope string

	if useProject {
		filePath = config.ProjectConfigPath()
		scope = "project"
	} else {
		var err error
		filePath, err = config.UserConfigPath()
		if err != nil {
			return fmt.Errorf("getting user config path: %w", err)
		}
		scope = "user"
	}

	value, found := getValueFromFile(filePath, keyPath)
	if !found {
		fmt.Fprintf(out, "%s: not set in %s config\n", key, scope)
		return nil
	}
	fmt.Fprintf(out, "%s: %s (from %s config)\n", key, value, scope)
	return nil
}

func getEffectiveValue(out io.Writer, key string, keyPath []string) error {
	// Check project first (higher priority)
	projectPath := config.ProjectConfigPath()
	if value, found := getValueFromFile(projectPath, keyPath); found {
		fmt.Fprintf(out, "%s: %s (from project config)\n", key, value)
		return nil
	}

	// Then check user config
	userPath, err := config.UserConfigPath()
	if err != nil {
		return fmt.Errorf("getting user config path: %w", err)
	}
	if value, found := getValueFromFile(userPath, keyPath); found {
		fmt.Fprintf(out, "%s: %s (from user config)\n", key, value)
		return nil
	}

	// Fall back to default
	schema, _ := config.GetKeySchema(key)
	fmt.Fprintf(out, "%s: %v (default)\n", key, schema.Default)
	return nil
}

func getValueFromFile(filePath string, keyPath []string) (string, bool) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", false
	}

	node := config.GetNestedValue(&root, keyPath)
	if node == nil {
		return "", false
	}
	return node.Value, true
}

func runConfigToggle(cmd *cobra.Command, args []string) error {
	key := args[0]
	out := cmd.OutOrStdout()

	// Validate key is a boolean
	schema, err := config.GetKeySchema(key)
	if err != nil {
		return formatUnknownKeyError(key)
	}
	if schema.Type != config.TypeBool {
		return fmt.Errorf("key %q is not a boolean (type: %s)", key, schema.Type)
	}

	filePath, scope, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}

	keyPath, err := config.ParseKeyPath(key)
	if err != nil {
		return fmt.Errorf("parsing key path: %w", err)
	}

	// Get current value
	currentValue := false
	if value, found := getValueFromFile(filePath, keyPath); found {
		currentValue = value == "true"
	}

	// Toggle and set new value
	newValue := !currentValue
	valueStr := "false"
	if newValue {
		valueStr = "true"
	}

	if err := config.SetConfigValue(filePath, key, valueStr); err != nil {
		return fmt.Errorf("setting config value: %w", err)
	}

	fmt.Fprintf(out, "Toggled %s: %t -> %t in %s config (%s)\n",
		key, currentValue, newValue, scope, filePath)
	return nil
}

func runConfigKeys(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	// Collect and sort keys
	keys := make([]string, 0, len(config.KnownKeys))
	for key := range config.KnownKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Fprintln(out, "Available configuration keys:")
	fmt.Fprintln(out)

	for _, key := range keys {
		schema := config.KnownKeys[key]
		typeInfo := schema.Type.String()
		if schema.Type == config.TypeEnum {
			typeInfo = fmt.Sprintf("enum (%s)", strings.Join(schema.AllowedValues, ", "))
		}
		fmt.Fprintf(out, "  %-40s %s\n", key, typeInfo)
		fmt.Fprintf(out, "    %s\n", schema.Description)
		fmt.Fprintln(out)
	}

	return nil
}

func resolveConfigPath(cmd *cobra.Command) (filePath, scope string, err error) {
	useUser, _ := cmd.Flags().GetBool("user")
	useProject, _ := cmd.Flags().GetBool("project")

	if useUser && useProject {
		return "", "", fmt.Errorf("--user and --project are mutually exclusive")
	}

	if useProject {
		projectPath := config.ProjectConfigPath()
		// Check if we're in a project directory
		if _, err := os.Stat(".autospec"); os.IsNotExist(err) {
			return "", "", fmt.Errorf("not in a project directory (no .autospec directory found)")
		}
		return projectPath, "project", nil
	}

	// Default to user config
	userPath, err := config.UserConfigPath()
	if err != nil {
		return "", "", fmt.Errorf("getting user config path: %w", err)
	}
	return userPath, "user", nil
}

func formatUnknownKeyError(key string) error {
	keys := make([]string, 0, len(config.KnownKeys))
	for k := range config.KnownKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fmt.Errorf("unknown configuration key: %q\n\nValid keys:\n  %s",
		key, strings.Join(keys, "\n  "))
}
