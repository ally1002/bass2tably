# bass2tably

Turns a bass recording into an ASCII bass tab in the terminal — either printed once, or in an interactive TUI that plays the audio back with a cursor running over the tab. Optionally writes a Standard MIDI File too, so riffs never get forgotten again.

Pure Go DSP (YIN pitch detection arbitrated by a harmonic spectrum, onsets from a broadband energy envelope unioned with high-frequency spectral flux), no machine learning. Audio playback uses miniaudio through cgo; non-WAV input is decoded with the system `ffmpeg`.

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
- Sixteenth notes are found completely up to ~160 BPM (94 ms per note): 16 of 16 attacks on a synthetic run, every one pitched. Past that the result turns erratic — 12 of 16 at 180 BPM (83 ms), 15 of 16 at 200 BPM (75 ms). Pitch is not the limit; every segment that is found still resolves a pitch at 75 ms. The 60 ms merge window is, because it discards a correct spectral onset whenever the broadband detector has already placed a late one within that distance.
- Notes closer together than 60 ms merge into one. That window is also what stops a single pluck registering as two notes, so it cannot simply be shortened.
- Low-frequency rolloff in the recording (phone mic, aggressive highpass) can drop onsets on the low B string.
- Tempo is assumed constant for the whole take.
- Percussive playing with a wide dynamic range (slap and pop, ghost notes) loses the quiet attacks. The onset threshold is computed once for the whole take, so loud slaps raise it above everything quieter: on a slap recording we find 100 of the 142 attacks an independent detector reports, the misses being loud enough to pass the level gate but too gradual to pass the global flux threshold. The spectral detector does not help here — measured against the same recording at equal precision it finds fewer attacks than the broadband one, not more.
- A note repeated at the same pitch on a string that is still ringing is now caught in most cases: 37 of the 38 attacks in `testdata/ahhintro.wav`, against 27 before, all of which now resolve a pitch and reach the tab. Long uniform runs are where the misses concentrate — a bar of eleven G1 eighths comes out as eight.
- A note whose fundamental is missing from the recording used to come out sharp, with nothing in the pipeline able to notice. The open B string at 12.43 s in `testdata/ahhintro.wav` carries no energy at 30.9 Hz at all — the lowest partial present is the octave — and YIN's difference function then settles on a lag 8 % short of the true period, returning 33.7 Hz (C1) at every window size from 3072 to 8192. A 16384-point spectrum is now consulted for every note: its partials are scored against candidate fundamentals, a candidate may drop an octave only when the odd harmonics the octave cannot explain are present in the spectrum, the spectral estimate overrules YIN when the two disagree by more than 50 cents, and it answers alone on a segment YIN declines to voice at all. That note reads B0, and the same phrase at the end of the full take — where YIN gives up on the B0 entirely — keeps it instead of dropping the note. The two estimators agree to within 27 cents on 101 of the 105 segments measured across the two long recordings, so the override is rare by construction and `shh.wav` output is unchanged byte for byte. It does not help where neither estimator has grounds for an answer: at 6.59 s in the same file two candidates a minor third apart explain the same number of partials, and the note comes out A1 where the player plays E1.
