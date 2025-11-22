module github.com/glive/cmd/glive

go 1.24

toolchain go1.24.10

require (
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/glive/core v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.0
	golang.org/x/term v0.19.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.design/x/clipboard v0.7.1 // indirect
	golang.org/x/exp/shiny v0.0.0-20250606033433-dcc06ee1d476 // indirect
	golang.org/x/image v0.28.0 // indirect
	golang.org/x/mobile v0.0.0-20250606033058-a2a15c67f36f // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/glive/core => ../../pkg/core

replace github.com/glive/interface => ../../pkg/interface
