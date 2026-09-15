package audio

import (
	"testing"
	"time"

	"bass2tably/internal/synth"
)

func TestPlaybackReachesEnd(t *testing.T) {
	const (
		bpm  = 100.0
		rate = 44100
	)
	samples := synth.RenderLead(synth.TestRiff, bpm, rate, 0)
	dur := float64(len(samples)) / float64(rate)

	p, err := New(samples, rate)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer p.Close()

	if p.Playing() {
		t.Fatal("Playing() = true before Play()")
	}
	if err := p.Play(); err != nil {
		t.Skipf("could not start playback: %v", err)
	}
	start := time.Now()

	last := -1.0
	deadline := time.Now().Add(time.Duration((dur + 1.0) * float64(time.Second)))
	for !p.Done() {
		if time.Now().After(deadline) {
			t.Fatalf("not Done() after %.1f s (PosSec %.2f, want %.2f)", dur+1.0, p.PosSec(), dur)
		}
		pos := p.PosSec()
		if pos < last {
			t.Fatalf("PosSec went backwards: %.4f after %.4f", pos, last)
		}
		last = pos
		time.Sleep(50 * time.Millisecond)
	}
	elapsed := time.Since(start).Seconds()
	if elapsed < dur-0.5 || elapsed > dur+0.5 {
		t.Errorf("reached Done() in %.2f s, want %.2f ± 0.5 s", elapsed, dur)
	}
	if got := p.PosSec(); got != dur {
		t.Errorf("PosSec at end = %.4f, want exactly %.4f (counter clamps at total)", got, dur)
	}

	if err := p.Pause(); err != nil {
		t.Fatal(err)
	}
	p.SeekSec(0)
	if p.Done() {
		t.Fatal("Done() = true after SeekSec(0)")
	}
	if got := p.PosSec(); got != 0 {
		t.Fatalf("PosSec after SeekSec(0) = %.4f, want 0", got)
	}
	if err := p.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if got := p.PosSec(); got <= 0 {
		t.Fatalf("PosSec = %.4f after restart, want > 0", got)
	}
	if err := p.Pause(); err != nil {
		t.Fatal(err)
	}
}

func TestSeekClamps(t *testing.T) {
	samples := make([]float64, 44100)
	p, err := New(samples, 44100)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer p.Close()

	p.SeekSec(-5)
	if got := p.PosSec(); got != 0 {
		t.Errorf("SeekSec(-5): PosSec = %.4f, want 0", got)
	}
	p.SeekSec(99)
	if got := p.PosSec(); got != 1 {
		t.Errorf("SeekSec(99): PosSec = %.4f, want 1", got)
	}
	if !p.Done() {
		t.Error("Done() = false after seeking to the end")
	}
}
