module github.com/glive/cmd/glive

go 1.24

toolchain go1.24.10

require (
	github.com/glive/core v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/glive/core => ../../pkg/core

replace github.com/glive/interface => ../../pkg/interface
