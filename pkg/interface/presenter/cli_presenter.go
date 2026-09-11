package presenter

import (
	"fmt"
	"io"

	"github.com/glive/domain/entities"
	"github.com/glive/interface/cli"
	"github.com/glive/usecase/project"
)

// CLIPresenter formats output for CLI
type CLIPresenter struct {
	writer io.Writer
}

// NewCLIPresenter creates a new CLI presenter
func NewCLIPresenter(writer io.Writer) CLIPresenter {
	return CLIPresenter{writer: writer}
}

// PresentSuccess presents a successful operation
func (p CLIPresenter) PresentSuccess(output *project.SetupProjectOutput) error {
	project := output.Project

	fmt.Fprintf(p.writer, "\n%s\n", cli.Success("Success! Your project is ready at:"))
	fmt.Fprintf(p.writer, "   %s\n\n", project.LocalPath().Value())
	fmt.Fprintf(p.writer, "%s\n", cli.Info("Next steps:"))
	fmt.Fprintf(p.writer, "   cd %s\n", project.LocalPath().Value())

	// Give specific instructions based on project type
	switch project.Type() {
	case entities.ProjectTypeNodeJS:
		fmt.Fprintf(p.writer, "   npm start  # or check package.json for start script\n")
	case entities.ProjectTypePython:
		fmt.Fprintf(p.writer, "   python main.py  # or check README for instructions\n")
	case entities.ProjectTypeGo:
		fmt.Fprintf(p.writer, "   go run .\n")
	default:
		fmt.Fprintf(p.writer, "   # Check README.md for how to run the project\n")
	}
	fmt.Fprintln(p.writer)

	// Show warnings if any
	if len(output.Warnings) > 0 {
		fmt.Fprintf(p.writer, "%s\n", cli.Warning("Warnings:"))
		for _, warning := range output.Warnings {
			fmt.Fprintf(p.writer, "   - %s\n", warning)
		}
		fmt.Fprintln(p.writer)
	}

	return nil
}

// PresentError presents an error with context and suggestions
func (p CLIPresenter) PresentError(err error) error {
	formatted := cli.FormatError(err)
	fmt.Fprintf(p.writer, "\n%s\n", formatted)
	return err
}
