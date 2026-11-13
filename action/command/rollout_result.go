package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bytebase/bytebase/action/result"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	// resultTimeout bounds reading the result, so that a slow server delays
	// the job's exit and never hangs it.
	resultTimeout = 5 * time.Minute
	// resultLogCalls is how many task run logs are read at once.
	resultLogCalls = 8
)

// collectResult reads the rollout's final state and the logs of the stage the
// run ended on. It returns nil when no rollout was created. A read that fails
// costs detail and is logged; it never fails the command.
func collectResult(w *world.World, client *client) *result.Result {
	if w.Rollout == nil {
		return nil
	}
	// The command's context is canceled when the job is.
	ctx, cancel := context.WithTimeout(context.Background(), resultTimeout)
	defer cancel()

	if rollout, err := client.getRollout(ctx, w.Rollout.GetName()); err != nil {
		w.Logger.Warn("failed to refresh the rollout for its result, using the last state seen", "error", err)
	} else {
		w.Rollout = rollout
	}

	in := result.Input{
		URL:         strings.TrimSuffix(w.URL, "/"),
		TargetStage: w.TargetStage,
		Rollout:     w.Rollout,
		Logs:        map[string]*v1pb.TaskRunLog{},
	}
	_, stage := result.DetailStage(w.Rollout, w.TargetStage)
	if stage == nil {
		return result.Build(in)
	}

	taskRuns, err := client.listTaskRunsUnder(ctx, stage.GetName()+"/tasks/-")
	if err != nil {
		w.Logger.Warn("failed to list task runs for the rollout result", "error", err)
		return result.Build(in)
	}
	in.TaskRuns = taskRuns

	var mu sync.Mutex
	var failed int
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(resultLogCalls)
	for _, runs := range result.GroupTaskRuns(taskRuns) {
		latest := runs[len(runs)-1]
		if !result.HasLog(latest) {
			continue
		}
		g.Go(func() error {
			log, err := client.getTaskRunLog(gctx, latest.GetName())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				return nil
			}
			in.Logs[latest.GetName()] = log
			return nil
		})
	}
	_ = g.Wait()
	if failed > 0 {
		w.Logger.Warn("failed to read some task run logs, their tasks are reported by status only", "count", failed)
	}
	releases := newReleaseReader(w, client)
	for _, task := range stage.GetTasks() {
		// The release's file order says what a stopped run never reached.
		if name := task.GetDatabaseUpdate().GetRelease(); name != "" && task.GetStatus() != v1pb.Task_DONE {
			releases.get(ctx, name)
		}
	}
	in.Sources = readSources(ctx, w, client, releases, result.NeededSources(stage, taskRuns, in.Logs))
	in.Releases = releases.read
	return result.Build(in)
}

// releaseReader reads each release once. A release that cannot be read is
// logged once and left out.
type releaseReader struct {
	w      *world.World
	client *client
	tried  map[string]bool
	read   map[string]*v1pb.Release
}

func newReleaseReader(w *world.World, client *client) *releaseReader {
	return &releaseReader{w: w, client: client, tried: map[string]bool{}, read: map[string]*v1pb.Release{}}
}

func (r *releaseReader) get(ctx context.Context, name string) *v1pb.Release {
	if !r.tried[name] {
		r.tried[name] = true
		release, err := r.client.getRelease(ctx, name)
		if err != nil {
			r.w.Logger.Warn("failed to read the release for the rollout result", "release", name, "error", err)
		} else {
			r.read[name] = release
		}
	}
	return r.read[name]
}

// readSources reads the text that logged byte ranges point into. A release
// file is read from the job's checkout when its hash matches the release's,
// which needs no permission beyond reading the release, and from its sheet
// otherwise.
func readSources(ctx context.Context, w *world.World, client *client, releases *releaseReader, refs []result.SourceRef) map[string][]byte {
	sources := map[string][]byte{}
	var missing int
	// Set once a sheet read fails, so that an identity that may not read
	// sheets is refused once and not once per file.
	var sheetErr error
	for _, ref := range refs {
		sheet := ref.Sheet
		if ref.Release != "" {
			sheet = ""
			for _, file := range releases.get(ctx, ref.Release).GetFiles() {
				if file.GetVersion() != ref.Version {
					continue
				}
				if content, ok := readCheckedOutFile(file); ok {
					sources[ref.Key()] = content
				} else {
					sheet = file.GetSheet()
				}
				break
			}
		}
		if _, ok := sources[ref.Key()]; ok {
			continue
		}
		if sheet == "" || sheetErr != nil {
			missing++
			continue
		}
		content, err := client.getSheetContent(ctx, sheet)
		if err != nil {
			sheetErr = err
			missing++
			continue
		}
		sources[ref.Key()] = content
	}
	if missing > 0 {
		w.Logger.Warn("the rollout result shows some commands without their statement: run the job in a checkout of the migration files, or let its identity read sheets (bb.sheets.get)", "files", missing, "error", sheetErr)
	}
	return sources
}

// readCheckedOutFile reads a release file from the working directory. It is
// used only when it is byte for byte what the release holds.
func readCheckedOutFile(file *v1pb.Release_File) ([]byte, bool) {
	if file.GetPath() == "" || file.GetSheetSha256() == "" {
		return nil, false
	}
	content, err := os.ReadFile(file.GetPath())
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(content)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), file.GetSheetSha256()) {
		return nil, false
	}
	return content, true
}
