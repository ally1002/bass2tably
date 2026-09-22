package tui

type stubPlayer struct {
	playing    bool
	pos        float64
	done       bool
	playErr    error
	pauseErr   error
	closed     bool
	seeks      []float64
	playCalls  int
	pauseCalls int
	speed      float64
	loopA      float64
	loopB      float64
	loopSet    bool
	loopClears int
	clickOn    bool
	clickBPM   float64
	clickPhase float64
	clickEvery int
}

func (s *stubPlayer) Play() error {
	s.playCalls++
	if s.playErr != nil {
		return s.playErr
	}
	s.playing = true
	return nil
}

func (s *stubPlayer) Pause() error {
	s.pauseCalls++
	if s.pauseErr != nil {
		return s.pauseErr
	}
	s.playing = false
	return nil
}

func (s *stubPlayer) Playing() bool { return s.playing }

func (s *stubPlayer) PosSec() float64 { return s.pos }

func (s *stubPlayer) SeekSec(t float64) {
	s.seeks = append(s.seeks, t)
	s.pos = t
}

func (s *stubPlayer) Done() bool { return s.done }

func (s *stubPlayer) Close() { s.closed = true }

func (s *stubPlayer) SetSpeed(f float64) { s.speed = f }

func (s *stubPlayer) SetLoop(startSec, endSec float64) {
	s.loopA, s.loopB, s.loopSet = startSec, endSec, true
}

func (s *stubPlayer) ClearLoop() {
	s.loopSet = false
	s.loopClears++
}

func (s *stubPlayer) SetClick(bpm, offsetSec float64, accentEvery int, on bool) {
	s.clickOn, s.clickBPM, s.clickPhase, s.clickEvery = on, bpm, offsetSec, accentEvery
}
