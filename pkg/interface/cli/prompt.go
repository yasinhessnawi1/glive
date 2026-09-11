package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Confirm prompts the user for a yes/no confirmation
func Confirm(message string, defaultYes bool) bool {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Printf("%s %s: ", message, hint)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// On error, return default
		return defaultYes
	}

	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// Select prompts the user to select from a list of options
func Select(message string, options []string) int {
	if len(options) == 0 {
		return -1
	}

	fmt.Println(message)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}

	defaultOption := 1
	fmt.Printf("Select [%d]: ", defaultOption)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// On error, return default
		return defaultOption - 1
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultOption - 1
	}

	choice, err := strconv.Atoi(response)
	if err != nil {
		fmt.Printf("Invalid selection. Using default option %d.\n", defaultOption)
		return defaultOption - 1
	}

	if choice < 1 || choice > len(options) {
		fmt.Printf("Selection out of range. Using default option %d.\n", defaultOption)
		return defaultOption - 1
	}

	return choice - 1
}

// Input prompts the user for text input
func Input(message string, defaultValue string) string {
	prompt := message
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]", message, defaultValue)
	}
	fmt.Printf("%s: ", prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	return response
}
