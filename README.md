# bass2tably

Turns a WAV recording of a bass riff into an ASCII bass tab in the terminal (optionally a Standard MIDI File too), so riffs never get forgotten again.

Pure Go, pure DSP (YIN pitch detection + energy-based onset detection), no machine learning, no cgo.

## Usage

```
bass2tably --bpm 90 --tuning BEADG --out riff.mid riff.wav
```

| Flag | Default | Meaning |
|---|---|---|
| `--bpm` | (required) | tempo of the recording in BPM |
| `--tuning` | `BEADG` | `BEADG`, `EADG`, `DADG` or `BEAD` |
| `--grid` | `16` | quantization grid: finest note value is an eighth (8) or sixteenth (16) |
| `--out` | (none) | write a MIDI file to this path; no MIDI is written if omitted |

Output: an ASCII tab (16 slots per measure, 4 measures per line) and a per-note list (`#1  0.02s  A1  string A fret 0  1/2 beat`). With `--out`, also a `.mid` file (960 TPQN, GM Electric Bass finger).

## Recording guidelines

- Record against a metronome at the BPM you pass with `--bpm` — quantization depends on it.
- Mono or stereo WAV, any common bit depth; channels are mixed down.
- Prefer a DI box or audio interface. Phone microphones roll off below ~100 Hz, which weakens the fundamentals of the low B and E strings and can make their note onsets undetectable.
- One note at a time (the pipeline is monophonic).

## Test signal generator

```
go run ./cmd/gentest --bpm 100 --out testriff.wav
```

Writes a synthetic riff with known pitches and timing (including low B0 and E1), used as ground truth by the test suite:

```
go test ./...
```

## Known limitations

- No chords or double stops — the pipeline is strictly monophonic.
- No slides, bends, or hammer-ons: they come out as separate notes or wrong pitches.
- Sixteenth notes above ~160 BPM leave too few analysis frames per note for reliable pitch.
- Notes shorter than ~60 ms merge into their neighbor.
- Low-frequency rolloff in the recording (phone mic, aggressive highpass) can drop onsets on the low B string.
- The string/fret assignment is a greedy heuristic (prefer low frets, minimize hand movement); it does not globally optimize the whole phrase.
