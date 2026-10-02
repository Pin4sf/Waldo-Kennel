package daemon

import (
	"context"
	"database/sql"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/bridgeactivation"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ownercommand"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/secretstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/activationrepo"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/bridgepersist"
	"log/slog"
	"net/http"
)

type bridgeWiring struct {
	handler    http.Handler
	controller *bridgeactivation.Controller
	dbs        []*sql.DB
}

func (b *bridgeWiring) close() {
	if b == nil {
		return
	}
	for _, db := range b.dbs {
		_ = db.Close()
	}
}

type notReadyFactory struct{}

func (notReadyFactory) Ready(context.Context) error { return bridgeactivation.ErrNotReady }
func (notReadyFactory) New(context.Context, string, domain.DeviceBridgeDevice, func(devicebridge.ConnectionState)) (bridgeactivation.Session, error) {
	return nil, bridgeactivation.ErrNotReady
}

// Configuration failure degrades only the optional bridge, never the daemon.
// The opener is injectable so flag-off proves zero storage access.
func wireBridge(dataDir string, getenv func(string) string, authority *ownercommand.BridgeAuthority, log *slog.Logger, open func(string) (*sql.DB, error)) *bridgeWiring {
	cfg, e := bridgeactivation.LoadLocalConfig(getenv)
	if e != nil {
		log.Warn("local Waldo bridge configuration invalid; bridge disabled")
		return nil
	}
	if !cfg.Enabled {
		return nil
	}
	if authority == nil {
		log.Warn("local Waldo bridge authority unavailable; bridge disabled")
		return nil
	}
	b := &bridgeWiring{}
	fail := func() *bridgeWiring {
		b.close()
		log.Warn("local Waldo bridge dependencies unavailable; bridge disabled")
		return nil
	}
	repoDB, e := open(dataDir)
	if e != nil {
		return fail()
	}
	b.dbs = append(b.dbs, repoDB)
	sessionDB, e := open(dataDir)
	if e != nil {
		return fail()
	}
	b.dbs = append(b.dbs, sessionDB)
	pairer, e := devicebridge.NewPairingCoordinator(bridgepersist.New(sessionDB), secretstore.NewDeviceKeyStore(dataDir), nil, log, cfg.Origin+devicebridge.RedeemPath)
	if e != nil {
		return fail()
	}
	c, e := bridgeactivation.New(cfg.OwnerID, cfg.Origin, activationrepo.New(repoDB), pairer, notReadyFactory{})
	if e != nil {
		return fail()
	}
	b.controller = c
	b.handler = bridgeactivation.Handler(c, func(r *http.Request) bool { return authority.Authenticate(r.Header.Get("Authorization")) })
	return b
}
