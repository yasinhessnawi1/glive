module github.com/glive/usecase

go 1.26.8

toolchain go1.26.8

require (
	github.com/glive/domain v0.0.0-00010101000000-000000000000
	github.com/glive/infrastructure v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace (
	github.com/glive/domain => ../domain
	github.com/glive/infrastructure => ../infrastructure
)
