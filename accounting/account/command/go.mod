module accounting/account/command

go 1.13

replace (
	accounting/event => ../../event
	accounting/util => ../../util
)

require (
	accounting/event v0.0.0-00010101000000-000000000000
	accounting/util v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.1
)
