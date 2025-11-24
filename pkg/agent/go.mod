module github.com/glive/agent

go 1.23.0

toolchain go1.24.10

require (
	github.com/glive/core v0.0.0
	github.com/glive/interface v0.0.0-00010101000000-000000000000
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/gofiber/websocket/v2 v2.2.1
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/fasthttp/websocket v1.5.3 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.4.6 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/glive/core => ../core
	github.com/glive/interface => ../interface
)
