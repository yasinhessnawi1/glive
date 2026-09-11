package controller

import (
	"fmt"
	"os"

	"github.com/glive/domain/errors"
	"github.com/glive/domain/values"
	"github.com/spf13/cobra"
)

// ArgValidator interface for validating command arguments
type ArgValidator interface {
	Validate(arg string) error
}

// ValidateArgs creates a cobra argument validator
func ValidateArgs(validators ...ArgValidator) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < len(validators) {
			return errors.NewUserError("MISSING_ARGS",
				fmt.Sprintf("Expected %d arguments, got %d", len(validators), len(args)))
		}

		for i, validator := range validators {
			if i >= len(args) {
				break
			}
			if err := validator.Validate(args[i]); err != nil {
				return err
			}
		}

		return nil
	}
}

// URLArg validates a repository URL argument
type URLArg struct{}

// Validate validates a URL argument
func (v URLArg) Validate(arg string) error {
	_, err := values.ParseRepoURL(arg)
	return err
}

// ModeArg validates an execution mode argument
type ModeArg struct{}

// Validate validates a mode argument
func (v ModeArg) Validate(arg string) error {
	validModes := map[string]bool{
		"auto":     true,
		"assisted": true,
		"manual":   true,
	}
	if !validModes[arg] {
		return errors.NewUserError("INVALID_MODE",
			"Mode must be 'auto', 'assisted', or 'manual'")
	}
	return nil
}

// PathArg validates a file path argument
type PathArg struct {
	MustExist   bool
	AllowedRoot string
}

// Validate validates a path argument
func (v PathArg) Validate(arg string) error {
	var path *values.SafePath
	var err error

	if v.AllowedRoot != "" {
		path, err = values.NewSafePath(arg, v.AllowedRoot)
	} else {
		// If no root specified, use the path itself as root
		path, err = values.NewSafePath(arg, arg)
	}

	if err != nil {
		return err
	}

	if v.MustExist {
		if _, err := os.Stat(path.Absolute()); os.IsNotExist(err) {
			return errors.NewUserError("PATH_NOT_FOUND",
				fmt.Sprintf("Path does not exist: %s", arg))
		}
	}

	return nil
}

// ConfigKeyArg validates a configuration key
type ConfigKeyArg struct{}

// Validate validates a config key argument
func (v ConfigKeyArg) Validate(arg string) error {
	validKeys := map[string]bool{
		"api-key":        true,
		"api-provider":   true,
		"workspace-dir":  true,
		"agent-port":     true,
		"max-concurrent": true,
		"default-mode":   true,
		"enable-sandbox": true,
	}
	if !validKeys[arg] {
		return errors.NewUserError("INVALID_CONFIG_KEY",
			fmt.Sprintf("Unknown configuration key: %s", arg))
	}
	return nil
}

// AnyArg accepts any non-empty string
type AnyArg struct{}

// Validate validates any non-empty argument
func (v AnyArg) Validate(arg string) error {
	if arg == "" {
		return errors.NewUserError("EMPTY_ARG", "Argument cannot be empty")
	}
	return nil
}
