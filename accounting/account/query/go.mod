module github.com/benjohns1/es-accounting/accounting/account/query

go 1.13

replace (
	github.com/benjohns1/es-accounting/event => ../../../event
	github.com/benjohns1/es-accounting/util => ../../../util
)

require (
	github.com/benjohns1/es-accounting/event v0.0.0-00010101000000-000000000000
	github.com/benjohns1/es-accounting/util v0.0.0-00010101000000-000000000000
)
