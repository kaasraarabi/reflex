package inbound

import (
	"bufio"
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/proxy"
	"github.com/xtls/xray-core/proxy/reflex"
	"github.com/xtls/xray-core/transport/internet/stat"
)

// Handler is the Step 1 inbound skeleton for the Reflex protocol.
type Handler struct {
	clients       []*protocol.MemoryUser
	fallback      *FallbackConfig
	seenNonces    map[[16]byte]int64
	nonceLifetime time.Duration
	nonceMu       sync.Mutex
}

// MemoryAccount stores Reflex user credentials in memory.
type MemoryAccount struct {
	ID     string
	Policy string
}

// Equals implements protocol.Account.
func (a *MemoryAccount) Equals(account protocol.Account) bool {
	reflexAccount, ok := account.(*MemoryAccount)
	if !ok {
		return false
	}
	return a.ID == reflexAccount.ID
}

// ToProto implements protocol.Account.
func (a *MemoryAccount) ToProto() proto.Message {
	return &reflex.Account{Id: a.ID}
}

// FallbackConfig stores fallback port configuration.
type FallbackConfig struct {
	Dest uint32
}

// Network implements proxy.Inbound.
func (h *Handler) Network() []net.Network {
	return []net.Network{net.Network_TCP}
}

// Process implements proxy.Inbound.
func (h *Handler) Process(ctx context.Context, network net.Network, conn stat.Connection, dispatcher routing.Dispatcher) error {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	peeked, err := peekForDetection(reader, reflexMinHandshakeSize)
	if err != nil {
		return h.handleFallback(ctx, reader, conn)
	}

	if h.isReflexHandshake(peeked) {
		if h.isReflexMagic(peeked) {
			return h.handleReflexMagic(ctx, reader, conn, dispatcher)
		}
		if h.isHTTPPostLike(peeked) {
			return h.handleReflexHTTP(ctx, reader, conn, dispatcher)
		}
		return h.handleFallback(ctx, reader, conn)
	}

	return h.handleFallback(ctx, reader, conn)
}

func init() {
	common.Must(common.RegisterConfig((*reflex.InboundConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*reflex.InboundConfig))
	}))
}

// New creates a new Step 1 Reflex inbound handler.
func New(ctx context.Context, config *reflex.InboundConfig) (proxy.Inbound, error) {
	handler := &Handler{
		clients:       make([]*protocol.MemoryUser, 0, len(config.Clients)),
		seenNonces:    make(map[[16]byte]int64),
		nonceLifetime: defaultNonceLifetime,
	}
	for _, client := range config.Clients {
		handler.clients = append(handler.clients, &protocol.MemoryUser{
			Email:   client.Id,
			Account: &MemoryAccount{ID: client.Id, Policy: client.Policy},
		})
	}
	if config.Fallback != nil {
		handler.fallback = &FallbackConfig{Dest: config.Fallback.Dest}
	}
	return handler, nil
}
