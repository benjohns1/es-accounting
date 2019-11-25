package main

import (
	"github.com/benjohns1/es-accounting/eventstore"
	"github.com/benjohns1/es-accounting/eventstore/repo"
	"github.com/benjohns1/es-accounting/eventstore/transport"
)

func main() {

	es := eventstore.EventStore{
		Repo:      repo.NewInMem(),
		Transport: transport.NewHTTP(),
	}

	es.Start()
}
