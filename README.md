# bass2tably

Turns a bass recording into an ASCII bass tab in the terminal — either printed once, or in an interactive TUI that plays the audio back with a cursor running over the tab. Optionally writes a Standard MIDI File too, so riffs never get forgotten again.

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
| `--tui` | off | open the TUI for the given file |
| `--theme` | `amber` | TUI colour theme: `amber`, `neon` or `ink` |
| `--ascii` | off | TUI: draw the tab with ASCII only, for terminals without box-drawing glyphs |

Input can be WAV, or any format `ffmpeg` reads (`.m4a`, `.mp3`, `.ogg`, `.flac`).

Output: an ASCII tab (16 slots per measure, 4 measures per line) and a per-note list
(`#1    0.02s  A1   string A fret 0   [also: E fret 5, B fret 10]  1/2 beat`).
With `--out`, also a `.mid` file (960 TPQN, GM Electric Bass finger).

## TUI

The viewer draws the tab with box-drawing lines: a note's sustain runs as `═` for its length in the string's colour, a note held across a bar line reopens the next measure as `(3)`, a ruler above each system numbers the measures, notes already behind the playhead are dimmed and the current column is highlighted. Under each system a stem row marks every attack with `│` and every rest with `·`, and one or two beam rows group the notes shorter than a beat within their beat, as engraved rhythm does: `└───┘` is a pair of eighths, `└─┴─┴─┘` four sixteenths, `└─` a lone short note. Under the tab sits a waveform of the take with the playhead on it, and below that a status band with the clock, measure and beat on the left and the tempo, tuning, grid, measures per line and theme on the right. The header shows only the file, the zoom, and whether the rhythm rows are on. Terminals without box-drawing glyphs get the plain ASCII tab with `--ascii`. Three themes ship, each with a light and a dark variant chosen from the terminal background: `amber` (warm, tube-amp glow), `neon` (one colour per string, like coated string sets) and `ink` (monochrome). Pick one with `--theme` or cycle with `t`.

The layout adapts: in the `page` layout measures per line fit the terminal width by default and can be forced with `[` and `]`; `v` switches to the `scroll` layout, one long system that follows the playhead sideways and keeps it a third of the way across the screen, the way a DAW does. The note list docks at the bottom or on the right and can be resized.

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
| `z` | cycle the cell width (zoom): 2, 3 or 4 columns per grid slot |
| `r` | show or hide the rhythm rows under each system |
| `v` | switch between the `page` layout (measures wrap to the width, for reading) and the `scroll` layout (one long system, the playhead pinned a third of the way in, for playing along) |
| `t` | next theme |
| `?` | expand the help footer |
| `esc` | back to the file picker |
| `q` | quit |

The file finder opens when no file is given. Start typing to fuzzy-filter every audio file under the current directory (four levels deep, five thousand files at most; hidden directories and `node_modules` are skipped), with the files you opened before listed first; `esc` clears the filter, `enter` opens the selection or descends into a directory, `backspace` (or `h` with an empty filter) goes up a level and `~` jumps home. At ninety columns or more a preview pane shows the selected WAV's sample rate, channels, bit depth and duration over a mini waveform; other formats are decoded with ffmpeg when opened. Recent files are remembered in `$XDG_STATE_HOME/bass2tably/recent` (`~/.local/state` by default). In the finder `ctrl+t` cycles the theme and `ctrl+c` quits, since letters go to the filter.

The mouse works too: click the waveform to seek, click a note to jump to it, click a row of the note list or the finder to select it (again to open it), and scroll with the wheel. Inside tmux the mouse reaches the pane only with `set -g mouse on` in `~/.tmux.conf`.

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
- Percussive playing with a wide dynamic range (slap and pop, ghost notes) loses the quiet attacks. The onset threshold is computed once for the whole take, so loud slaps raise it above everything quieter: on a slap recording we find 100 of the 142 attacks an independent detector reports, the misses being loud enough to pass the level gate but too gradual to pass the global flux threshold. The spectral detector does not help here — measured against the same recording at equal precision it finds fewer attacks than the broadband one, not more.
- A note repeated at the same pitch on a string that is still ringing is now caught in most cases: 37 of the 38 attacks in `testdata/ahhintro.wav`, against 27 before, all of which now resolve a pitch and reach the tab. Long uniform runs are where the misses concentrate — a bar of eleven G1 eighths comes out as eight.
- A note whose fundamental is missing from the recording used to come out sharp, with nothing in the pipeline able to notice. The open B string at 12.43 s in `testdata/ahhintro.wav` carries no energy at 30.9 Hz at all — the lowest partial present is the octave — and YIN's threshold scan then settles on a lag 8 % short of the true period, 33.7 Hz (C1), because that dip is deeper than the true one in every frame of the note. YIN no longer chooses. Every local minimum of its difference function below a ceiling is kept as a candidate, the note's spectrum (a Hann window of up to 16384 points over the sustain) contributes every partial divided by its first ten harmonic numbers, and one cost ranks the union: a two-way mismatch that charges a candidate for the harmonics it predicts and cannot find and for the partials it cannot explain, with harmonics placed at `h·f₀·√(1+B·h²)` for string stiffness and magnitudes square-root compressed so four loud partials cannot outvote seven quiet ones. The decision is made once per note over all its frames; YIN's parabolic lag then supplies the precise frequency when it agrees with the winner. Bounds come from the tuning: nothing below a semitone under the lowest open string or above fret 24 is scored, so on a four-string `--tuning EADG` the same B0 comes out as its octave B1 rather than as a fret that does not exist. Measured over the 190 segments of the test corpus, the ranking agrees with the previous release on every note it got right — `shh.wav` output is byte for byte unchanged — and decides both open-B notes (12.43 s in the intro cut, 0.73 s in `testdata/ahhoutro.wav`) by a margin above the corpus median; all six B0s of the full take survive. The published weights of the mismatch cost had to change: with the paper's 1:3 balance between the two terms the octave wins on an *absent* fundamental, and the measured plateau puts the terms near 1:1.6. What remains wrong is one note: at 6.59 s in `testdata/ahhintro.wav` a short E1 whose partials form no harmonic series under any window tried is ranked C1 by the cost, G1 by YIN and A1 by a plain harmonic count, each by a margin far below the corpus's tenth percentile; a least-squares oracle run over the whole corpus ranks the E1 fourth there too.
