package presenter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/glive/usecase/project"
)

// JSONPresenter formats output as JSON
type JSONPresenter struct {
	writer io.Writer
}

// NewJSONPresenter creates a new JSON presenter
func NewJSONPresenter(writer io.Writer) JSONPresenter {
	return JSONPresenter{writer: writer}
}

// PresentSuccess presents a successful operation as JSON
func (p JSONPresenter) PresentSuccess(output *project.SetupProjectOutput) error {
	response := map[string]interface{}{
		"success": true,
		"project": map[string]interface{}{
			"id":         output.Project.ID().Value(),
			"name":       output.Project.Name(),
			"url":        output.Project.URL().String(),
			"local_path": output.Project.LocalPath().Value(),
			"type":       string(output.Project.Type()),
			"status":     string(output.Project.Status()),
		},
		"commands_count": len(output.Commands),
		"warnings":      output.Warnings,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	fmt.Fprintln(p.writer, string(data))
	return nil
}

// PresentError presents an error as JSON
func (p JSONPresenter) PresentError(err error) error {
	response := map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	fmt.Fprintln(p.writer, string(data))
	return err
}

