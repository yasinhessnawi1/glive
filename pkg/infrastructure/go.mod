module github.com/glive/infrastructure

go 1.21

require (
	github.com/glive/core v0.0.0-00010101000000-000000000000
	github.com/glive/domain v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.17.0
	golang.org/x/sys v0.15.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/glive/core => ../core

replace github.com/glive/domain => ../domain
