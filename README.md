# bass2tably

Turns a bass recording into an ASCII bass tab in the terminal — either printed once, or in an interactive TUI that plays the audio back with a cursor running over the tab. Optionally writes a Standard MIDI File too, so riffs never get forgotten again.

Pure Go DSP (YIN pitch detection + energy-based onset detection), no machine learning. Audio playback uses miniaudio through cgo; non-WAV input is decoded with the system `ffmpeg`.

## Usage

```
bass2tably                        # TUI: pick a file, then read and play the tab
bass2tably --tui riff.wav         # TUI, straight to the tab viewer
bass2tably --bpm 90 riff.wav      # print the tab and note list and exit
```

| Flag | Default | Meaning |
|---|---|---|
| `--bpm` | estimate | tempo of the recording in BPM; `0` estimates it from the note onsets |
| `--tuning` | `BEADG` | `BEADG`, `EADG`, `DADG` or `BEAD` |
| `--grid` | `16` | quantization grid: finest note value is an eighth (8) or sixteenth (16) |
| `--position` | `0` | bias fret assignment toward this neck position |
| `--out` | (none) | write a MIDI file to this path; no MIDI is written if omitted |
| `--tui` | off | open the TUI for the given file |

Input can be WAV, or any format `ffmpeg` reads (`.m4a`, `.mp3`, `.ogg`, `.flac`).

Output: an ASCII tab (16 slots per measure, 4 measures per line) and a per-note list
(`#1    0.02s  A1   string A fret 0   [also: E fret 5, B fret 10]  1/2 beat`).
With `--out`, also a `.mid` file (960 TPQN, GM Electric Bass finger).

## TUI keys

| Key | Action |
|---|---|
| `space` | play / pause |
| `shift+←` `shift+→` or `,` `.` | seek one measure back / forward |
| `n` | toggle the note list panel |
| `↑/↓` `j/k` | scroll the tab; with the note panel open, move the note selection (and seek there) |
| `enter` | play from the selected note |
| `←/→` `h/l` | scroll the tab horizontally |
| `g` `G` | jump to top / bottom |
| `?` | expand the help footer |
| `esc` | back to the file picker |
| `q` | quit |

The tab scrolls itself to keep the playback cursor visible while playing; scrolling by hand while paused is never overridden. If no audio device is available the tab still opens, with a notice that playback is disabled.

## Recording guidelines

- Record against a metronome, and pass its tempo with `--bpm` when you know it — quantization depends on it, and the estimator only sees the onsets it detected.
- Mono or stereo, any common bit depth; channels are mixed down.
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

Tests that need an audio device skip themselves when none is available.

## Known limitations

- No chords or double stops — the pipeline is strictly monophonic.
- No slides, bends, or hammer-ons: they come out as separate notes or wrong pitches.
- Sixteenth notes above ~160 BPM leave too few analysis frames per note for reliable pitch.
- Notes shorter than ~60 ms merge into their neighbor.
- Low-frequency rolloff in the recording (phone mic, aggressive highpass) can drop onsets on the low B string.
- Tempo is assumed constant for the whole take.
- Percussive playing with a wide dynamic range (slap and pop, ghost notes) loses the quiet attacks. The onset threshold is computed once for the whole take, so loud slaps raise it above everything quieter: on a slap recording we find 100 of the 142 attacks an independent detector reports, the misses being loud enough to pass the level gate but too gradual to pass the global flux threshold.
- A note repeated at the same pitch and a sustained note that re-triggers the onset detector are not reliably told apart, so some repeats are merged.
