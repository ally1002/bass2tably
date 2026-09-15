package audio

import (
	"encoding/binary"
	"math"
	"sync/atomic"

	"github.com/gen2brain/malgo"
)

type Player struct {
	ctx     *malgo.AllocatedContext
	dev     *malgo.Device
	rate    int
	frames  atomic.Int64
	total   int64
	samples []float64
}

func New(samples []float64, rate int) (*Player, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	p := &Player{
		ctx:     ctx,
		rate:    rate,
		total:   int64(len(samples)),
		samples: samples,
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Playback)
	cfg.Playback.Format = malgo.FormatF32
	cfg.Playback.Channels = 1
	cfg.SampleRate = uint32(rate)
	cfg.PeriodSizeInMilliseconds = 20

	// SeekSec storing between this load and store can lose one ~20 ms
	// period of the seek; accepted, see PLAN.md C.1.
	onData := func(out, _ []byte, frameCount uint32) {
		pos := p.frames.Load()
		n := int64(frameCount)
		for i := int64(0); i < n; i++ {
			var v float32
			if idx := pos + i; idx < p.total {
				v = float32(p.samples[idx])
			}
			binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
		}
		if pos += n; pos > p.total {
			pos = p.total
		}
		p.frames.Store(pos)
	}

	dev, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: onData})
	if err != nil {
		ctx.Uninit()
		ctx.Free()
		return nil, err
	}
	p.dev = dev
	return p, nil
}

func (p *Player) Play() error {
	return p.dev.Start()
}

func (p *Player) Pause() error {
	return p.dev.Stop()
}

func (p *Player) Playing() bool {
	return p.dev.IsStarted()
}

func (p *Player) PosSec() float64 {
	return float64(p.frames.Load()) / float64(p.rate)
}

func (p *Player) SeekSec(t float64) {
	f := int64(t * float64(p.rate))
	if f < 0 {
		f = 0
	}
	if f > p.total {
		f = p.total
	}
	p.frames.Store(f)
}

func (p *Player) Done() bool {
	return p.frames.Load() >= p.total
}

func (p *Player) Close() {
	p.dev.Uninit()
	p.ctx.Uninit()
	p.ctx.Free()
}
