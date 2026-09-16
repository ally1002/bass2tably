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
