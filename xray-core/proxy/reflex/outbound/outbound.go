package outbound

import (
	"context"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/proxy"
	"github.com/xtls/xray-core/proxy/reflex"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet"
)

// Handler is the Step 1 outbound skeleton for the Reflex protocol.
type Handler struct {
	config *reflex.OutboundConfig
}

// Process implements proxy.Outbound.
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbounds := session.OutboundsFromContext(ctx)
	ob := outbounds[len(outbounds)-1]
	ob.Name = "reflex"
	_ = common.Interrupt(link.Writer)
	return errors.New("reflex outbound step1 skeleton: handshake/session not implemented")
}

func init() {
	common.Must(common.RegisterConfig((*reflex.OutboundConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*reflex.OutboundConfig))
	}))
}

// New creates a new Step 1 Reflex outbound handler.
func New(ctx context.Context, config *reflex.OutboundConfig) (proxy.Outbound, error) {
	return &Handler{config: config}, nil
}
