package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
	"bass2tably/internal/wavio"
)

const cacheVersion = 1

type cacheKey struct {
	Path    string
	Size    int64
	ModTime int64
	Opts    Options
	Version int
}

type cached struct {
	SampleRate   int
	DurationSec  float64
	Peak         float64
	BPM          float64
	BPMEstimated bool
	GridDiv      int
	Tuning       []int
	Notes        []music.Note
	Frets        []tab.Fretting
	Skipped      []Skipped
}

type Summary struct {
	Notes     int
	BPM       float64
	Estimated bool
}

func keyFor(path string, opts Options) (cacheKey, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return cacheKey{}, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return cacheKey{}, err
	}
	return cacheKey{Path: abs, Size: fi.Size(), ModTime: fi.ModTime().UnixNano(), Opts: opts, Version: cacheVersion}, nil
}

func cacheFile(k cacheKey) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%+v", k)))
	return filepath.Join(dir, "bass2tably", hex.EncodeToString(sum[:8])+".gob"), nil
}

func summaryFile(gobPath string) string {
	return gobPath[:len(gobPath)-len(".gob")] + ".sum"
}

func writeGob(path string, v any) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readGob(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func Store(path string, opts Options, res *Result) {
	k, err := keyFor(path, opts)
	if err != nil {
		return
	}
	file, err := cacheFile(k)
	if err != nil {
		return
	}
	c := cached{
		SampleRate:   res.SampleRate,
		DurationSec:  res.DurationSec,
		Peak:         res.Peak,
		BPM:          res.BPM,
		BPMEstimated: res.BPMEstimated,
		GridDiv:      res.GridDiv,
		Tuning:       res.Tuning,
		Notes:        res.Notes,
		Frets:        res.Frets,
		Skipped:      res.Skipped,
	}
	if err := writeGob(file, c); err != nil {
		return
	}
	writeGob(summaryFile(file), Summary{Notes: len(res.Notes), BPM: res.BPM, Estimated: res.BPMEstimated})
}

func load(path string, opts Options) (*cached, bool) {
	k, err := keyFor(path, opts)
	if err != nil {
		return nil, false
	}
	file, err := cacheFile(k)
	if err != nil {
		return nil, false
	}
	var c cached
	if err := readGob(file, &c); err != nil {
		return nil, false
	}
	return &c, true
}

func Peek(path string, opts Options) (Summary, bool) {
	k, err := keyFor(path, opts)
	if err != nil {
		return Summary{}, false
	}
	file, err := cacheFile(k)
	if err != nil {
		return Summary{}, false
	}
	var s Summary
	if err := readGob(summaryFile(file), &s); err != nil {
		return Summary{}, false
	}
	return s, true
}

func AnalyzeCached(path string, opts Options) (*Result, error) {
	if c, ok := load(path, opts); ok {
		if res, err := fromCache(path, c); err == nil {
			return res, nil
		}
	}
	res, err := Analyze(path, opts)
	if err != nil {
		return nil, err
	}
	Store(path, opts, res)
	return res, nil
}

func fromCache(path string, c *cached) (*Result, error) {
	clip, err := wavio.LoadAny(path)
	if err != nil {
		return nil, err
	}
	if clip.SampleRate != c.SampleRate || len(clip.Samples) == 0 {
		return nil, fmt.Errorf("cache: sample rate or length changed")
	}
	tabStr, layout := tab.RenderLayout(c.Notes, c.Frets, c.Tuning, c.GridDiv)
	res := &Result{
		Path:         path,
		SampleRate:   c.SampleRate,
		DurationSec:  c.DurationSec,
		Peak:         c.Peak,
		Samples:      clip.Samples,
		BPM:          c.BPM,
		BPMEstimated: c.BPMEstimated,
		GridDiv:      c.GridDiv,
		Tuning:       c.Tuning,
		Notes:        c.Notes,
		Frets:        c.Frets,
		Skipped:      c.Skipped,
		Tab:          tabStr,
		Layout:       layout,
	}
	if err := res.Valid(); err != nil {
		return nil, err
	}
	return res, nil
}
