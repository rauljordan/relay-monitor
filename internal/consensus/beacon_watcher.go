package consensus

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/beaconapi"
	"github.com/protolambda/eth2api/client/validatorapi"
	"github.com/protolambda/zrnt/eth2/beacon/bellatrix"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	consensustypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	"github.com/r3labs/sse/v2"
	"github.com/ralexstokes/relay-monitor/internal/cache"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField("prefix", "consensus")
)

// BeaconWatcher is solely responsible for listening to beacon chain events, slots, epochs,
// and populating caches that are necessary for other processes in the project.
type BeaconWatcher struct {
	consensusCache    *cache.ConsensusDataCache
	epochClock        *helpers.EpochTicker
	genesis           uint64
	beaconAPIEndpoint string
}

type Option = func(b *BeaconWatcher) error

func WithCache(c *cache.ConsensusDataCache) Option {
	return func(b *BeaconWatcher) error {
		b.consensusCache = c
		return nil
	}
}

func WithEndpoint(endpoint string) Option {
	return func(b *BeaconWatcher) error {
		b.beaconAPIEndpoint = endpoint
		return nil
	}
}

func WithGenesisTime(genesis uint64) Option {
	return func(b *BeaconWatcher) error {
		b.genesis = genesis
		return nil
	}
}

func New(opts ...Option) (*BeaconWatcher, error) {
	bw := &BeaconWatcher{}
	for _, opt := range opts {
		if err := opt(bw); err != nil {
			return nil, err
		}
	}
	return bw, nil
}

func (bw *BeaconWatcher) Start(ctx context.Context) {
	bw.epochClock = helpers.NewEpochTicker(time.Unix(int64(bw.genesis), 0), 12*32)
	client := &eth2api.Eth2HttpClient{
		Addr: bw.beaconAPIEndpoint,
		Cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 128,
			},
			Timeout: 12 * time.Second,
		},
		Codec: eth2api.JSONCodec{},
	}
	// Listen for new chain head events and put data in a cache.
	ch := make(chan *types.Coordinate, 1)
	go func(c chan *types.Coordinate) {
		if err := bw.subscribeBeaconHeads(ctx, c); err != nil {
			log.Error(err)
		}
	}(ch)
	// Process head events...
	go func() {
		for head := range ch {
			blockID := eth2api.BlockIdSlot(head.Slot)
			var signedBeaconBlock eth2api.VersionedSignedBeaconBlock
			exists, err := beaconapi.BlockV2(ctx, client, blockID, &signedBeaconBlock)
			if !exists {
				log.Error("Could not get block")
				continue
			}
			if err != nil {
				log.Error("Could not get block")
				continue
			}
			bellatrixBlock, ok := signedBeaconBlock.Data.(*bellatrix.SignedBeaconBlock)
			if !ok {
				log.Error("Could not get block")
				continue
			}
			executionHash := types.Hash(bellatrixBlock.Message.Body.ExecutionPayload.BlockHash)
			log.Infof("%#x", executionHash)
			_ = executionHash
		}
	}()
	// Process epoch events...
	// Every epoch, refresh proposer duties and put them in a cache.
	for epoch := range bw.epochClock.C() {
		var proposerDuties eth2api.DependentProposerDuty
		syncing, err := validatorapi.ProposerDuties(ctx, client, common.Epoch(epoch), &proposerDuties)
		if syncing {
			log.Error("Syncing..")
			continue
		} else if err != nil {
			log.Error("Error occurred..")
			continue
		}
		for _, duty := range proposerDuties.Data {
			log.WithFields(logrus.Fields{
				"slot":           duty.Slot,
				"publicKey":      duty.Pubkey.String(),
				"validatorIndex": duty.ValidatorIndex,
			})
		}

		// TODO handle reorgs, etc.
		//c.proposerCacheMutex.Lock()
		//for _, duty := range proposerDuties.Data {
		//	c.proposerCache[consensustypes.Slot(duty.Slot)] = ValidatorInfo{
		//		publicKey: types.PublicKey(duty.Pubkey),
		//		index:     uint64(duty.ValidatorIndex),
		//	}
		//}
		//c.proposerCacheMutex.Unlock()
	}
}

type headEvent struct {
	Slot  string     `json:"slot"`
	Block types.Root `json:"block"`
}

func (bw *BeaconWatcher) subscribeBeaconHeads(ctx context.Context, ch chan<- *types.Coordinate) error {
	sseClient := sse.NewClient(bw.beaconAPIEndpoint + "/eth/v1/events?topics=head")
	return sseClient.SubscribeRaw(func(msg *sse.Event) {
		if ctx.Err() != nil {
			return
		}
		var event headEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Warnf("could not unmarshal `head` node event: %v", err)
			return
		}
		slot, err := strconv.Atoi(event.Slot)
		if err != nil {
			log.Warnf("could not unmarshal slot from `head` node event: %v", err)
			return
		}
		head := &types.Coordinate{
			Slot: consensustypes.Slot(slot),
			Root: event.Block,
		}
		ch <- head
	})
}
