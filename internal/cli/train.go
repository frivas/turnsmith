package cli

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/terminal"
	"github.com/frivas/turnsmith/internal/training"
	"github.com/frivas/turnsmith/internal/view"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c CLI) trainCommand(ctx context.Context, args []string, out io.Writer) error {
	o, e := options(args, map[string]string{"file": "", "data-dir": "", "mode": "balanced", "seed": "", "case": "", "status": "", "input": "", "output": "", "token": "", "json": "false", "list": "false", "replace": "false", "export-progress": "false", "import-progress": "", "color": "auto", "color-depth": "auto"}, "json", "list", "replace", "export-progress")
	if e != nil {
		return e
	}
	if o.yes("help") {
		return commandHelp(out, "train")
	}
	action := "practice"
	if len(o.pos) > 1 {
		return app.Invalid("train accepts one action")
	}
	if len(o.pos) == 1 {
		action = o.pos[0]
	}
	aliases := 0
	if o.yes("list") {
		action = "list"
		aliases++
	}
	if o.yes("export-progress") {
		action = "export"
		aliases++
	}
	if o.seen["import-progress"] {
		action = "import"
		o.values["input"] = o.values["import-progress"]
		aliases++
	}
	if aliases > 1 || (aliases > 0 && len(o.pos) > 0) {
		return app.Invalid("choose one training action")
	}
	if o.yes("json") && action == "practice" {
		action = "list"
	}
	switch action {
	case "list", "practice", "status", "export", "import", "recover", "lock", "unlock":
	default:
		return app.Invalid("unknown train action %q", action)
	}
	allowed := map[string]string{
		"list":     "file data-dir json list",
		"practice": "file data-dir mode seed color color-depth json",
		"status":   "file data-dir case status",
		"export":   "data-dir output file export-progress",
		"import":   "data-dir input import-progress replace",
		"recover":  "data-dir", "lock": "data-dir json", "unlock": "data-dir token",
	}
	for flag := range o.seen {
		if flag != "help" && !strings.Contains(" "+allowed[action]+" ", " "+flag+" ") {
			return app.Invalid("--%s is not valid for train %s", flag, action)
		}
	}
	if action == "practice" {
		if e := c.checkTerminal(out); e != nil {
			return e
		}
	}
	var collection training.Collection
	if action == "list" || action == "practice" || action == "status" {
		if o.values["file"] == "" {
			return app.Invalid("--file CASES.json is required")
		}
		file, e := os.Open(o.values["file"])
		if e != nil {
			return e
		}
		collection, e = training.LoadCollection(file)
		_ = file.Close()
		if e != nil {
			return invalid(e)
		}
	}
	store, e := training.NewStore(o.values["data-dir"])
	if e != nil {
		return e
	}
	switch action {
	case "list":
		progress, e := store.Progress(ctx, collection)
		if e != nil {
			return e
		}
		if o.yes("json") {
			return writeJSON(out, struct {
				Collection training.Collection     `json:"collection"`
				Progress   []training.CaseProgress `json:"progress"`
			}{collection, progress})
		}
		for i, item := range collection.Cases {
			if _, e := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%d attempts\n", item.ID, item.Puzzle, item.Name, progress[i].Status, len(progress[i].Attempts)); e != nil {
				return e
			}
		}
		return nil
	case "status":
		if o.values["case"] == "" || o.values["status"] == "" {
			return app.Invalid("status requires --case ID and --status new|learning|known")
		}
		if e := store.SetStatus(ctx, collection, o.values["case"], training.Status(o.values["status"])); e != nil {
			return invalid(e)
		}
		_, e = fmt.Fprintln(out, "Case status saved.")
		return e
	case "export":
		if o.values["output"] == "" {
			return store.ExportJSON(ctx, out)
		}
		if o.values["file"] != "" && trainSamePath(o.values["file"], o.values["output"]) {
			return app.Invalid("progress output must differ from the collection file")
		}
		var data strings.Builder
		if e := store.ExportJSON(ctx, &data); e != nil {
			return e
		}
		return trainWriteFile(o.values["output"], []byte(data.String()))
	case "import":
		path := o.values["input"]
		if path == "" {
			return app.Invalid("import requires --input PROGRESS.json or --import-progress PATH")
		}
		file, e := os.Open(path)
		if e != nil {
			return e
		}
		defer file.Close()
		if e := store.ImportJSON(ctx, file, o.yes("replace")); e != nil {
			return e
		}
		_, e = fmt.Fprintln(out, "Training progress imported.")
		return e
	case "recover":
		if e := store.Recover(ctx); e != nil {
			return e
		}
		_, e = fmt.Fprintln(out, "Validated training backup restored; previous primary bytes preserved.")
		return e
	case "lock":
		owner, e := store.LockInfo()
		if e != nil {
			return e
		}
		return writeJSON(out, owner)
	case "unlock":
		if o.values["token"] == "" {
			return app.Invalid("unlock requires --token from train lock; stop every process using the store first")
		}
		if e := store.BreakLock(o.values["token"]); e != nil {
			return e
		}
		_, e = fmt.Fprintln(out, "Training lock removed.")
		return e
	}
	policy := training.Policy(o.values["mode"])
	switch policy {
	case training.Balanced, training.LearningFirst, training.Slowest, training.Sequential:
	default:
		return app.Invalid("--mode must be balanced, learning, slowest or sequential")
	}
	var seed uint64
	if o.seen["seed"] {
		seed, e = random.ParseSeed(o.values["seed"])
	} else {
		seed, e = random.Seed(c.Service.Entropy)
	}
	if e != nil {
		return invalid(e)
	}
	style, e := c.viewOptions(o)
	if e != nil {
		return e
	}
	model := trainModel{ctx: ctx, store: store, collection: collection, policy: policy, rng: random.Derive(seed, "training", "selection", "1", 0), style: style, seed: seed}
	if e := model.next(); e != nil {
		return e
	}
	return terminal.Run(ctx, c.Input, out, terminal.Options{Tick: 100 * time.Millisecond}, model.handle)
}

type trainModel struct {
	ctx                  context.Context
	store                *training.Store
	collection           training.Collection
	policy               training.Policy
	rng                  random.Source
	style                view.Options
	seed                 uint64
	item                 training.Case
	setup                []alg.Move
	state                cubestate.State
	status               training.Status
	started              time.Time
	elapsed              time.Duration
	pending, saved, hint bool
	message              string
	setupPage            int
}

func (m *trainModel) next() error {
	item, e := m.store.Next(m.ctx, m.collection, m.policy, m.rng)
	if e != nil {
		return e
	}
	setup, e := item.Setup()
	if e != nil {
		return e
	}
	size, e := item.Size()
	if e != nil {
		return e
	}
	states, e := cubestate.Replay(size, setup)
	if e != nil {
		return e
	}
	progress, e := m.store.Progress(m.ctx, m.collection)
	if e != nil {
		return e
	}
	for _, p := range progress {
		if p.CaseID == item.ID {
			m.status = p.Status
		}
	}
	m.item, m.setup, m.state = item, setup, states[len(states)-1]
	m.started = time.Time{}
	m.elapsed = 0
	m.pending = false
	m.saved = false
	m.hint = false
	m.setupPage = 0
	m.message = "Follow the setup, then press Space to start."
	return nil
}
func (m *trainModel) record(now time.Time, success bool) error {
	if !m.pending {
		return nil
	}
	if e := m.store.Record(m.ctx, m.collection, m.item.ID, training.Attempt{At: now.UTC(), DurationNS: int64(m.elapsed), Hint: m.hint, Success: success}); e != nil {
		return e
	}
	m.pending = false
	m.saved = true
	if success {
		m.message = "Successful attempt saved. Space repeats; n selects next."
	} else {
		m.message = "Unsuccessful attempt saved. Space repeats; n selects next."
	}
	return nil
}
func (m *trainModel) stop(now time.Time) {
	m.elapsed = now.Sub(m.started)
	if m.elapsed < 0 {
		m.elapsed = 0
	}
	m.started = time.Time{}
	m.pending = true
	m.message = "Confirm correctness: y/Space successful, d unsuccessful."
}
func (m *trainModel) handle(event terminal.Event) (terminal.Frame, error) {
	minimumHeight := max(24, 3*(m.state.Size+1)+14)
	tooSmall := event.Width < 48 || event.Height < minimumHeight
	smallText := fmt.Sprintf("Training needs 48 columns and %d rows. Resize to start.\nSpace stops/confirms; d unsuccessful; y successful; q discards.\n", minimumHeight)
	active := !m.started.IsZero() || m.pending
	safeKey := event.Key == terminal.KeySpace || event.Key == terminal.KeyEnter || (event.Key == terminal.KeyRune && strings.ContainsRune("yYdD", event.Rune))
	if tooSmall && (!active || !safeKey) {
		return terminal.Frame{Text: smallText + m.message}, nil
	}
	pages := trainPages(alg.Tokens(m.setup), max(8, event.Width-24))
	m.setupPage = min(m.setupPage, m.pageCount(event.Width)-1)
	now := event.At
	if now.IsZero() {
		now = time.Now()
	}
	running := !m.started.IsZero()
	switch event.Key {
	case terminal.KeySpace, terminal.KeyEnter:
		if running {
			m.stop(now)
		} else if m.pending {
			if e := m.record(now, true); e != nil {
				return terminal.Frame{}, e
			}
		} else {
			if m.saved {
				m.hint = false
			}
			m.saved = false
			m.elapsed = 0
			m.started = now
			m.message = "Timing. Space stops; d stops and records unsuccessful."
		}
	case terminal.KeyNext:
		if event.Rune == 0 {
			m.setupPage = min(m.setupPage+1, m.pageCount(event.Width)-1)
			break
		}
		if running || m.pending {
			m.message = "Stop and record this attempt before selecting the next case."
		} else if e := m.next(); e != nil {
			m.message = e.Error()
		}
	case terminal.KeyPrevious:
		m.setupPage = max(0, m.setupPage-1)
	case terminal.KeyHome:
		m.setupPage = 0
	case terminal.KeyRune:
		switch event.Rune {
		case 'h', 'H':
			m.hint = true
			m.message = "Solution revealed; this attempt will record hint use."
		case 'y', 'Y':
			if e := m.record(now, true); e != nil {
				return terminal.Frame{}, e
			}
		case 'd', 'D':
			if running {
				m.stop(now)
			}
			if e := m.record(now, false); e != nil {
				return terminal.Frame{}, e
			}
		case 'l', 'L', 'k', 'K', '0':
			status := training.Learning
			if event.Rune == 'k' || event.Rune == 'K' {
				status = training.Known
			} else if event.Rune == '0' {
				status = training.New
			}
			if e := m.store.SetStatus(m.ctx, m.collection, m.item.ID, status); e != nil {
				return terminal.Frame{}, e
			}
			m.status = status
			m.message = "Status saved: " + string(status)
		}
	}
	if tooSmall {
		return terminal.Frame{Text: smallText + m.message}, nil
	}
	pages = trainPages(alg.Tokens(m.setup), max(8, event.Width-24))
	m.setupPage = min(m.setupPage, m.pageCount(event.Width)-1)
	elapsed := m.elapsed
	if !m.started.IsZero() {
		elapsed = now.Sub(m.started)
		if elapsed < 0 {
			elapsed = 0
		}
	}
	solution := "Press h to reveal your algorithm."
	if m.hint {
		hintPages := trainPages(alg.Tokens(alg.Inverse(m.setup)), max(8, event.Width-24))
		solution = fmt.Sprintf("Hint page %d/%d: %s", min(m.setupPage+1, len(hintPages)), len(hintPages), hintPages[min(m.setupPage, len(hintPages)-1)])
	}
	style := m.style
	style.Width = event.Width
	text := fmt.Sprintf("%s | %s (%s) | %s\nMode %s | selection seed %d\nSetup page %d/%d: %s\n%s%sTime %.3fs\n%s\n%s\nSpace start/stop/confirm | y success | d unsuccessful\nh hint | n next | arrows setup pages | q quit\nl learning | k known | 0 new\nQuitting discards an unfinished attempt.\n", m.collection.Name, m.item.Name, m.item.Puzzle, m.status, m.policy, m.seed, min(m.setupPage+1, len(pages)), len(pages), pages[min(m.setupPage, len(pages)-1)], view.Net(m.state, style), solution+"\n", elapsed.Seconds(), m.message, view.Orientation(m.state.Size))
	return terminal.Frame{Text: text}, nil
}

func trainPages(tokens []string, width int) []string {
	pages := []string{}
	line := ""
	for _, token := range tokens {
		if len(line)+len(token)+1 > width && line != "" {
			pages = append(pages, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += token
	}
	if line != "" {
		pages = append(pages, line)
	}
	if len(pages) == 0 {
		pages = []string{"(identity)"}
	}
	return pages
}

func trainSamePath(a, b string) bool {
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	if aa == bb {
		return true
	}
	as, ae := os.Stat(a)
	bs, be := os.Stat(b)
	return ae == nil && be == nil && os.SameFile(as, bs)
}
func trainWriteFile(path string, data []byte) error {
	file, e := os.CreateTemp(filepath.Dir(path), ".training-export-*")
	if e != nil {
		return e
	}
	name := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(name) }()
	if e := file.Chmod(0600); e != nil {
		return e
	}
	if _, e := file.Write(data); e != nil {
		return e
	}
	if e := file.Sync(); e != nil {
		return e
	}
	if e := file.Close(); e != nil {
		return e
	}
	return os.Rename(name, path)
}

func (m *trainModel) pageCount(width int) int {
	count := len(trainPages(alg.Tokens(m.setup), max(8, width-24)))
	if m.hint {
		count = max(count, len(trainPages(alg.Tokens(alg.Inverse(m.setup)), max(8, width-24))))
	}
	return count
}
