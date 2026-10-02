package bridgeruntime

import (
	"context"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"math"
	"sync/atomic"
	"time"
)

const (
	baseBackoff      = 2 * time.Second
	maxBackoff       = 5 * time.Minute
	stableConnection = 60 * time.Second
)

type Session struct {
	client     *devicebridge.Client
	connect    func(context.Context) error
	fence      Fence
	owner      string
	device     domain.DeviceBridgeDeviceID
	generation int64
	clock      func() time.Time
	sleep      func(context.Context, time.Duration) error
	jitter     func() float64
	rejected   *atomic.Bool
}

func (s *Session) Run(ctx context.Context) error {
	delay := baseBackoff
	for ctx.Err() == nil {
		if s.fence.VerifyClaim(ctx, s.owner, s.device, s.generation) != nil {
			return nil
		}
		started := s.clock()
		e := s.connect(ctx)
		if errors.Is(e, devicebridge.ErrAuthenticationRejected) || s.rejected != nil && s.rejected.Load() || ctx.Err() != nil {
			return nil
		}
		if s.clock().Sub(started) >= stableConnection {
			delay = baseBackoff
		}
		random := s.jitter()
		if math.IsNaN(random) || random < 0 {
			random = 0
		}
		if random > 1 {
			random = 1
		}
		pause := time.Duration(float64(delay) * (0.8 + 0.4*random))
		if pause > maxBackoff {
			pause = maxBackoff
		}
		if s.sleep(ctx, pause) != nil {
			return nil
		}
		delay *= 2
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	return nil
}
func interruptibleSleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
