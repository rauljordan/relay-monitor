package api

type API struct {
	subscribers []chan uint64
}

func (a *API) SubscribeMEVData(ch chan uint64) error {
	a.subscribers = append(a.subscribers, ch)
	return nil
}
