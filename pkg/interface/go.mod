module github.com/glive/interface

go 1.21

require (
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/glive/domain v0.0.0-00010101000000-000000000000
	github.com/glive/infrastructure v0.0.0-00010101000000-000000000000
	github.com/glive/usecase v0.0.0-00010101000000-000000000000
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/google/uuid v1.5.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/term v0.19.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.17.0 // indirect
)

replace (
	github.com/glive/domain => ../domain
	github.com/glive/infrastructure => ../infrastructure
	github.com/glive/usecase => ../usecase
)
