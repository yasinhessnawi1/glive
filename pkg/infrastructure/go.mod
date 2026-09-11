module github.com/glive/infrastructure

go 1.26.8

require (
	github.com/glive/core v0.0.0-00010101000000-000000000000
	github.com/glive/domain v0.0.0-00010101000000-000000000000
	github.com/gofiber/fiber/v2 v2.52.0
	golang.org/x/crypto v0.17.0
	golang.org/x/sys v0.33.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.4.6 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/glive/core => ../core

replace github.com/glive/domain => ../domain
