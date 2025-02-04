module github.com/mulesoft-anypoint/muletracker-cli

go 1.23.5

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/mulesoft-anypoint/muletracker-cli/cmd v0.0.1
	github.com/mulesoft-anypoint/muletracker-cli/config v0.0.1
)


replace github.com/mulesoft-anypoint/muletracker-cli/cmd => ./cmd
replace github.com/mulesoft-anypoint/muletracker-cli/config => ./config
