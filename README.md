# bass2tably

Turns a bass recording into an ASCII bass tab in the terminal — either printed once, or in an interactive TUI that plays the audio back with a cursor running over the tab. Optionally writes a Standard MIDI File or a Guitar Pro 5 file too, so riffs never get forgotten again.

Pure Go DSP (pitch from YIN's period candidates ranked by a two-way harmonic mismatch against the note's spectrum, onsets from a broadband energy envelope unioned with high-frequency spectral flux), no machine learning. Audio playback uses miniaudio through cgo; non-WAV input is decoded with the system `ffmpeg`.

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
| `--gp` | (none) | write a Guitar Pro 5 file to this path; nothing is written if omitted |
| `--tui` | off | open the TUI for the given file |
| `--theme` | `amber` | TUI colour theme: `amber`, `neon` or `ink` |

Input can be WAV, or any format `ffmpeg` reads (`.m4a`, `.mp3`, `.ogg`, `.flac`).

Output: an ASCII tab (16 slots per measure, 4 measures per line) and a per-note list
(`#1    0.02s  A1   string A fret 0   [also: E fret 5, B fret 10]  1/2 beat`).
With `--out`, also a `.mid` file (960 TPQN, GM Electric Bass finger).
With `--gp`, also a Guitar Pro 5 file: one bass track, 4/4, the tab's strings and frets, for import into Guitar Pro, TuxGuitar, MuseScore or Songsterr's Upload.

## TUI

The viewer colours each string, marks the playback position as a column through the tab, and keeps a transport bar (time, progress, measure and beat) under the header. Three themes ship, each with a light and a dark variant chosen from the terminal background: `amber` (warm, tube-amp glow), `neon` (one colour per string, like coated string sets) and `ink` (monochrome). Pick one with `--theme` or cycle with `t`.

The layout adapts: measures per line fit the terminal width by default and can be forced with `[` and `]`; the note list docks at the bottom or on the right and can be resized.

| Key | Action |
|---|---|
| `space` | play / pause |
| `shift+←` `shift+→` or `,` `.` | seek one measure back / forward |
| `n` | toggle the note list panel |
| `N` | dock the note list at the bottom or on the right |
| `+` `-` | grow / shrink the note list |
| `↑/↓` `j/k` | scroll the tab; with the note panel open, move the note selection (and seek there) |
| `enter` | play from the selected note |
| `←/→` `h/l` | scroll the tab horizontally |
| `g` `G` | jump to top / bottom |
| `[` `]` | fewer / more measures per line |
| `0` | fit measures per line to the terminal width again |
| `t` | next theme |
| `?` | expand the help footer |
| `esc` | back to the file picker |
| `q` | quit |

In the file picker, `~` jumps to the home directory and `h` goes up one level.

The tab scrolls itself to keep the playback cursor visible while playing; scrolling by hand while paused is never overridden. If no audio device is available the tab still opens, with a notice in the transport bar that playback is disabled.

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
- The Guitar Pro 5 export writes one bass track, one voice, 4/4 throughout, an integer tempo, and plain notes, rests, dots and ties on a binary grid: no tuplets, no pickup bar, no techniques, no dynamics beyond one fixed level, and no metadata beyond the title. A note that starts before the previous one has ended is dropped and counted in the `--gp` summary line. Songsterr's Upload accepts the files as written and keeps tuning, strings, frets, ties and dotted values (measured 2026-09-22 on `testdata/minimal.gp5` and `testdata/shh.gp5`).
- Percussive playing with a wide dynamic range (slap and pop, ghost notes) loses the quiet attacks. The onset threshold is computed once for the whole take, so loud slaps raise it above everything quieter: on a slap recording we find 100 of the 142 attacks an independent detector reports, the misses being loud enough to pass the level gate but too gradual to pass the global flux threshold. The spectral detector does not help here — measured against the same recording at equal precision it finds fewer attacks than the broadband one, not more.
- A note repeated at the same pitch on a string that is still ringing is now caught in most cases: 37 of the 38 attacks in `testdata/ahhintro.wav`, against 27 before, all of which now resolve a pitch and reach the tab. Long uniform runs are where the misses concentrate — a bar of eleven G1 eighths comes out as eight.
- A note whose fundamental is missing from the recording used to come out sharp, with nothing in the pipeline able to notice. The open B string at 12.43 s in `testdata/ahhintro.wav` carries no energy at 30.9 Hz at all — the lowest partial present is the octave — and YIN's threshold scan then settles on a lag 8 % short of the true period, 33.7 Hz (C1), because that dip is deeper than the true one in every frame of the note. YIN no longer chooses. Every local minimum of its difference function below a ceiling is kept as a candidate, the note's spectrum (a Hann window of up to 16384 points over the sustain) contributes every partial divided by its first ten harmonic numbers, and one cost ranks the union: a two-way mismatch that charges a candidate for the harmonics it predicts and cannot find and for the partials it cannot explain, with harmonics placed at `h·f₀·√(1+B·h²)` for string stiffness and magnitudes square-root compressed so four loud partials cannot outvote seven quiet ones. The decision is made once per note over all its frames; YIN's parabolic lag then supplies the precise frequency when it agrees with the winner. Bounds come from the tuning: nothing below a semitone under the lowest open string or above fret 24 is scored, so on a four-string `--tuning EADG` the same B0 comes out as its octave B1 rather than as a fret that does not exist. Measured over the 190 segments of the test corpus, the ranking agrees with the previous release on every note it got right — `shh.wav` output is byte for byte unchanged — and decides both open-B notes (12.43 s in the intro cut, 0.73 s in `testdata/ahhoutro.wav`) by a margin above the corpus median; all six B0s of the full take survive. The published weights of the mismatch cost had to change: with the paper's 1:3 balance between the two terms the octave wins on an *absent* fundamental, and the measured plateau puts the terms near 1:1.6. What remains wrong is one note: at 6.59 s in `testdata/ahhintro.wav` a short E1 whose partials form no harmonic series under any window tried is ranked C1 by the cost, G1 by YIN and A1 by a plain harmonic count, each by a margin far below the corpus's tenth percentile; a least-squares oracle run over the whole corpus ranks the E1 fourth there too.
