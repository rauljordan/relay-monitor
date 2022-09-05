package processor

import "fmt"

type db interface {
	readBid() error
}

type MEVDataProvider interface {
	SubscribeMEVData(ch chan uint64) error
}

type Processor struct {
	db            db
	dataProviders []MEVDataProvider
}

func (p *Processor) Start() {
	// Subscribes to feed of bids, blocks, registrations, and more from data providers.
	// Does not care who the providers are. This helps us abstract procesing from ingestion.
	// Examples of providers are the fault submission API and the monitor process in the codebase.
	mevDataRecv := make(chan uint64, 100)
	defer close(mevDataRecv)
	for _, provider := range p.dataProviders {
		if err := provider.SubscribeMEVData(mevDataRecv); err != nil {
			panic(err)
		}
	}
	for data := range mevDataRecv {
		fmt.Println("Received MEV data", data)
	}
	// TODO: Add with policies to figure out what we want to track.
	// Allow registration of custom policies by exposing an endpoint and give it a unique pooint.
}
