package monitor

type Monitor struct {
	subscribers []chan uint64
}

func (m *Monitor) Start() {

}

func (m *Monitor) SubscribeMEVData(ch chan uint64) error {
	m.subscribers = append(m.subscribers, ch)
	return nil
}
