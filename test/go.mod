module github.com/trebuchet-org/treb-cli/test

go 1.24

toolchain go1.24.3

require (
	github.com/google/go-cmp v0.7.0
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/stretchr/testify v1.10.0
	github.com/trebuchet-org/treb-cli v0.0.0
)

require golang.org/x/term v0.31.0 // indirect

replace github.com/trebuchet-org/treb-cli => ../

require (
	github.com/briandowns/spinner v1.23.2
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
