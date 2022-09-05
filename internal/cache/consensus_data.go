package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

var (
	maxBlocks      = 4096
	cacheMissCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consensus_data_cache_miss",
		Help: "The number of check point state requests that aren't present in the cache.",
	})
	cacheHitCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consensus_data_cache_hit",
		Help: "The number of check point state requests that are present in the cache.",
	})
)

// ConsensusDataCache --
type ConsensusDataCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewConsensusDataCache --
func NewConsensusDataCache() *ConsensusDataCache {
	return &ConsensusDataCache{
		cache: lruwrpr.New(maxBlocks),
	}
}

func (c *ConsensusDataCache) ProposerDuties(epoch types.Epoch) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(epoch)
	if exists && item != nil {
		cacheHitCount.Inc()
		// TODO: Copy...
		return item.([]uint64), nil
	}
	cacheMissCount.Inc()
	return nil, nil
}

func (c *ConsensusDataCache) AddProposerDuties(epoch types.Epoch, duties []uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(epoch, duties)
}
