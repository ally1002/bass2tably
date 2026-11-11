# bass2tably

Turns a bass recording into a bass tab in the terminal — printed once as ASCII, or in an interactive TUI that draws sustains, ties and rhythm beams, plays the audio back under a cursor, and loops, clicks and slows a section for practice. Optionally writes a Standard MIDI File or a Guitar Pro 5 file too, so riffs never get forgotten again.

Pure Go DSP (pitch from YIN's period candidates ranked by a two-way harmonic mismatch against the note's spectrum, onsets from a broadband energy envelope unioned with high-frequency spectral flux), no machine learning. Audio playback uses miniaudio through cgo; non-WAV input is decoded with the system `ffmpeg`.

## Install

Needs Go 1.26.2 or newer (from `go.mod`) and a C compiler, because playback goes through miniaudio via cgo (`github.com/gen2brain/malgo`). `ffmpeg` is only needed for input that is not WAV.

```
git clone https://github.com/ally1002/bass2tably.git
cd bass2tably
go install .
```

This puts a `bass2tably` binary in `$(go env GOPATH)/bin`.

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
| `--ascii` | off | TUI: draw the tab with ASCII only, for terminals without box-drawing glyphs |

Input can be WAV, or any format `ffmpeg` reads (`.m4a`, `.mp3`, `.ogg`, `.flac`).

Output: an ASCII tab (16 slots per measure, 4 measures per line) and a per-note list
(`#1    0.02s  A1   string A fret 0   [also: E fret 5, B fret 10]  1/2 beat`).
With `--out`, also a `.mid` file (960 TPQN, GM Electric Bass finger).
With `--gp`, also a Guitar Pro 5 file: one bass track, 4/4, the tab's strings and frets, for import into Guitar Pro, TuxGuitar, MuseScore or Songsterr's Upload.

## TUI

`bass2tably` with no file opens the finder; `bass2tably --tui riff.wav` goes straight to the viewer.

### The tab

The viewer draws a Songsterr-shaped tab from a typed model of the analysis, not from the command-line text. String lines are box-drawing glyphs (`─ │ ║`); a note's sustain runs as `═` for its length in the string's colour; a note held across a bar line reopens the next measure as `(3)`; a ruler above each system numbers the measures; notes already behind the playhead are dimmed and the current column is highlighted. Under each system a stem row marks every attack with `│` and the start of every rest with `·`, and one or two beam rows group the notes shorter than a beat within their beat, as engraved rhythm does: `└───┘` is a pair of eighths, `└─┴─┴─┘` four sixteenths, `└─` a lone short note. `r` hides the rhythm rows and `z` cycles the cell width between 2, 3 and 4 columns per grid slot (a two-digit fret always gets at least 3).

Below the tab sits a waveform of the take with the playhead on it, then a status band with the clock, measure and beat on the left and the tempo, tuning, grid, layout and theme on the right. Terminals without box-drawing glyphs get plain `-` and `|` with `--ascii`. Three themes ship, each with a light and a dark variant chosen from the terminal background: `amber` (warm, tube-amp glow), `neon` (one colour per string, like coated string sets) and `ink` (monochrome). Pick one with `--theme` or cycle with `t`.

### Two layouts

In the `page` layout measures wrap to the terminal width; `[` and `]` force the count and `0` fits it again. `v` switches to the `scroll` layout: one long system that follows the playhead sideways and keeps it a third of the way across the screen, the way a DAW does, for playing along. The tab scrolls itself to keep the cursor visible while playing; scrolling by hand while paused is never overridden. The note list (`n`) docks at the bottom or on the right (`N`) and can be resized (`+` `-`).

### The finder

Start typing to fuzzy-filter every audio file under the current directory — four levels deep, five thousand files at most, hidden directories and `node_modules` skipped — with the files you opened before listed first. `esc` clears the filter, `enter` opens the selection or descends into a directory, `backspace` (or `h` with an empty filter) goes up a level and `~` jumps home. At ninety columns or more a preview pane shows the selected WAV's sample rate, channels, bit depth and duration over a mini waveform; other formats are decoded with ffmpeg when opened. Since letters go to the filter, in the finder `ctrl+t` cycles the theme and `ctrl+c` quits.

Recent files are remembered in `$XDG_STATE_HOME/bass2tably/recent` (`~/.local/state` by default). Analysis results are cached in `$XDG_CACHE_HOME/bass2tably` (`~/.cache` by default), keyed by the file's path, size and modification time and by the options used, so reopening a take is instant and the finder marks files already analysed with the current options as `✓ 64 notes · 124.5 bpm`. Delete that directory to clear the cache; the command-line output never reads it.

### Mouse

Click the waveform to seek, click a note to jump to it, click a row of the note list or the finder to select it (again to open it), and scroll with the wheel. Inside tmux the mouse reaches the pane only with `set -g mouse on` in `~/.tmux.conf`.

### Practice tools

Loop, metronome and speed live inside the audio callback, so they are sample-exact. `a` sets the loop start at the current measure and `b` the loop end after the current measure (they swap if the end lands first); `x` clears it; the ruler and the waveform shade the looped measures and the loop wraps at the bar line with no gap. `m` clicks a metronome on the grid the tab was quantized to, starting from the first onset and accented every four beats. `<` and `>` step the speed through 0.5×, 0.75× and 1×. Speed is a plain rate change, like a tape machine: 0.5× plays an octave lower, and the status band says so.

If no audio device is available the tab still opens, with a notice in the status band that playback is disabled.

### Export

`e` opens a small dialog: tick Guitar Pro 5 (the same file `--gp` writes) and/or MIDI with `space` on the highlighted row (nothing is ticked by default), edit the destination — prefilled next to the input, without extension; `~` works — then `enter` writes and `esc` cancels. The status band reports what was written for three seconds.

### Keys

Every action is declared once in one table; the help footer (`?`), the command palette (`ctrl+p` or `:`, type to filter, `enter` runs) and this list are all generated from it. A key listed twice acts on the note list while it is open and on the tab otherwise.

| Key | Action |
|---|---|
| `space` | Play / pause |
| `enter` | Play from the selected note |
| `shift+←/,` | Seek one measure back |
| `shift+→/.` | Seek one measure forward |
| `↑/k` | Select the previous note |
| `↓/j` | Select the next note |
| `g` | Select the first note |
| `G` | Select the last note |
| `↑/k` | Scroll the tab up |
| `↓/j` | Scroll the tab down |
| `←/h` | Scroll the tab left |
| `→/l` | Scroll the tab right |
| `g` | Scroll to the top |
| `G` | Scroll to the bottom |
| `n` | Toggle the note list |
| `N` | Dock the note list at the bottom or on the right |
| `+` | Grow the note list |
| `-` | Shrink the note list |
| `[` | Fewer measures per line |
| `]` | More measures per line |
| `0` | Fit measures per line to the width |
| `z` | Cycle the cell width (zoom) |
| `r` | Toggle the rhythm rows |
| `v` | Switch between the page and scroll layouts |
| `a` | Set loop start at this measure |
| `b` | Set loop end after this measure |
| `x` | Clear the loop |
| `m` | Toggle the metronome |
| `<` | Slower playback |
| `>` | Faster playback |
| `e` | Export to Guitar Pro 5 / MIDI… |
| `ctrl+p` | Command palette |
| `t` | Next theme |
| `?` | Toggle the full help |
| `esc` | Back to the finder |
| `q` | Quit |

## How it works

1. Decoding. A `.wav` file is read directly and its channels are averaged to mono. Any other extension is converted by `ffmpeg` to a temporary 44.1 kHz mono 16-bit WAV first.
2. Onsets. Two detectors run over the whole take. One is a broadband RMS envelope (512-sample windows, 256-sample hop) whose rises are picked when they exceed the mean plus 2.2 standard deviations and the envelope is above 5% of its maximum. The other is a high-frequency spectral flux: frame-to-frame increases in log magnitude, weighted by bin number. The broadband onsets are kept as they are, and a spectral onset is added only if it is at least 60 ms from every onset already found. A note ends at the next onset, or earlier if the envelope drops below 10% of the note's peak.
3. Pitch. The first 10 ms of each note are skipped. YIN runs over frames three periods of the lowest allowed frequency long, and every local minimum of its difference function becomes a candidate, alongside candidates taken from the note's spectral partials. A two-way mismatch cost ranks them, with harmonics stretched for string stiffness, and YIN's estimate supplies the exact frequency when it agrees with the winner. Only pitches from a semitone below the lowest open string up to fret 24 on the highest one are considered.
4. Tempo. Without `--bpm`, the tempo is the value between 55 and 200 BPM whose sixteenth-note grid best fits the gaps between onsets, with a preference for tempos near 110.
5. Quantizing. Each note's start, measured from the first note, is rounded to the grid (`--grid`), and its length is rounded too and cut off where the next note starts.
6. Fretting. A dynamic program picks a string and fret for every note so that the hand moves as little as possible along the neck, then across strings, pulled toward `--position`.
7. Output. The same notes and frets feed the ASCII tab and note list, the TUI, the MIDI writer (960 ticks per quarter note) and the Guitar Pro 5 writer.

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
- The rhythm rows show attacks, rests and beams only: no dotted values, tuplets or techniques (slides, hammer-ons, ghost notes). The Guitar Pro export carries the dots and ties.
- Sixteenth notes are found completely up to ~160 BPM (94 ms per note): 16 of 16 attacks on a synthetic run, every one pitched. Past that the result turns erratic — 12 of 16 at 180 BPM (83 ms), 15 of 16 at 200 BPM (75 ms). Pitch is not the limit; every segment that is found still resolves a pitch at 75 ms. The 60 ms merge window is, because it discards a correct spectral onset whenever the broadband detector has already placed a late one within that distance.
- Notes closer together than 60 ms merge into one. That window is also what stops a single pluck registering as two notes, so it cannot simply be shortened.
- Low-frequency rolloff in the recording (phone mic, aggressive highpass) can drop onsets on the low B string.
- Tempo is assumed constant for the whole take.
- The first detected note is always placed on beat one of the first bar, so a riff that starts with a pickup is shifted.
- The tempo estimator only searches 55 to 200 BPM. Outside that range, pass `--bpm`.
- Only the four tunings listed under `--tuning` are available.
- The Guitar Pro 5 export writes one bass track, one voice, 4/4 throughout, an integer tempo, and plain notes, rests, dots and ties on a binary grid: no tuplets, no pickup bar, no techniques, no dynamics beyond one fixed level, and no metadata beyond the title. A note that starts before the previous one has ended is dropped and counted in the `--gp` summary line. Songsterr's Upload accepts the files as written and keeps tuning, strings, frets, ties and dotted values (measured 2026-09-22 on `testdata/minimal.gp5` and `testdata/looped-phrase.gp5`).
- Percussive playing with a wide dynamic range (slap and pop, ghost notes) loses the quiet attacks. The onset threshold is computed once for the whole take, so loud slaps raise it above everything quieter: on a slap recording we find 100 of the 142 attacks an independent detector reports, the misses being loud enough to pass the level gate but too gradual to pass the global flux threshold. The spectral detector does not help here — measured against the same recording at equal precision it finds fewer attacks than the broadband one, not more.
- A note repeated at the same pitch on a string that is still ringing is now caught in most cases: 37 of the 38 attacks in `testdata/four-bar-repeats.wav`, against 27 before, all of which now resolve a pitch and reach the tab. Long uniform runs are where the misses concentrate — a bar of eleven G1 eighths comes out as eight.
- A note whose fundamental is missing from the recording used to come out sharp, with nothing in the pipeline able to notice. The open B string at 12.43 s in `testdata/four-bar-repeats.wav` carries no energy at 30.9 Hz at all — the lowest partial present is the octave — and YIN's threshold scan then settles on a lag 8 % short of the true period, 33.7 Hz (C1), because that dip is deeper than the true one in every frame of the note. YIN no longer chooses. Every local minimum of its difference function below a ceiling is kept as a candidate, the note's spectrum (a Hann window of up to 16384 points over the sustain) contributes every partial divided by its first ten harmonic numbers, and one cost ranks the union: a two-way mismatch that charges a candidate for the harmonics it predicts and cannot find and for the partials it cannot explain, with harmonics placed at `h·f₀·√(1+B·h²)` for string stiffness and magnitudes square-root compressed so four loud partials cannot outvote seven quiet ones. The decision is made once per note over all its frames; YIN's parabolic lag then supplies the precise frequency when it agrees with the winner. Bounds come from the tuning: nothing below a semitone under the lowest open string or above fret 24 is scored, so on a four-string `--tuning EADG` the same B0 comes out as its octave B1 rather than as a fret that does not exist. Measured over the 190 segments of the test corpus, the ranking agrees with the previous release on every note it got right — `looped-phrase.wav` output is byte for byte unchanged — and decides both open-B notes (12.43 s in the four-bar cut, 0.73 s in `testdata/missing-fundamental.wav`) by a margin above the corpus median; all six B0s of the full take survive. The published weights of the mismatch cost had to change: with the paper's 1:3 balance between the two terms the octave wins on an *absent* fundamental, and the measured plateau puts the terms near 1:1.6. What remains wrong is one note: at 6.59 s in `testdata/four-bar-repeats.wav` a short E1 whose partials form no harmonic series under any window tried is ranked C1 by the cost, G1 by YIN and A1 by a plain harmonic count, each by a margin far below the corpus's tenth percentile; a least-squares oracle run over the whole corpus ranks the E1 fourth there too.

## License

MIT, see [LICENSE](LICENSE).
