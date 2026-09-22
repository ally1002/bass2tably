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
	params  atomic.Pointer[params]
	scratch []float32
	frac    float64
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
		scratch: make([]float32, rate/10),
	}
	p.params.Store(&params{speed: 1})

	cfg := malgo.DefaultDeviceConfig(malgo.Playback)
	cfg.Playback.Format = malgo.FormatF32
	cfg.Playback.Channels = 1
	cfg.SampleRate = uint32(rate)
	cfg.PeriodSizeInMilliseconds = 20

	// SeekSec storing between this load and store can lose one ~20 ms
	// period of the seek; accepted, see PLAN.md C.1.
	onData := func(out, _ []byte, frameCount uint32) {
		n := int(frameCount)
		if cap(p.scratch) < n {
			p.scratch = make([]float32, n)
		}
		buf := p.scratch[:n]
		pos := p.frames.Load()
		pos, p.frac = mix(buf, p.samples, pos, p.frac, p.params.Load())
		for i, v := range buf {
			binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
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

func (p *Player) update(f func(*params)) {
	next := *p.params.Load()
	f(&next)
	p.params.Store(&next)
}

func (p *Player) SetSpeed(f float64) {
	if f <= 0 {
		return
	}
	p.update(func(q *params) { q.speed = f })
}

func (p *Player) SetLoop(startSec, endSec float64) {
	a := int64(startSec * float64(p.rate))
	b := int64(endSec * float64(p.rate))
	a = max(a, 0)
	b = min(b, p.total)
	if b <= a {
		p.ClearLoop()
		return
	}
	p.update(func(q *params) { q.loopA, q.loopB = a, b })
}

func (p *Player) ClearLoop() {
	p.update(func(q *params) { q.loopA, q.loopB = 0, 0 })
}

func (p *Player) SetClick(bpm, offsetSec float64, accentEvery int, on bool) {
	var c *clickTrack
	if on {
		c = newClick(p.rate, bpm, offsetSec, accentEvery)
	}
	p.update(func(q *params) { q.click = c })
}
