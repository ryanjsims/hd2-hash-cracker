package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hellflame/argparse"
	"github.com/xypwn/gocl/cl-3.1"

	"github.com/xypwn/hd2-hash-cracker/cli"
	"github.com/xypwn/hd2-hash-cracker/hash"
	"github.com/xypwn/hd2-hash-cracker/pattern"
	pcl "github.com/xypwn/hd2-hash-cracker/pattern/cl"
	"github.com/xypwn/hd2-hash-cracker/util"
)

type cracker struct {
	ctx    context.Context
	err    error
	ctxErr error

	mu sync.Mutex
	// Guarded by mu //
	msgBuf            []string
	newHashes         []string // newly cracked hashes
	newUniqueHashes   map[string]struct{}
	lastStr           string
	extraStatusStr    string
	triesCnt          int
	triesPerSecondBuf util.RingBuf[float64]
	// End           //
}

func runCracker(ctx context.Context, patternSrc []byte, targetHashes []uint64) (newHashes []string, err error) {
	c := &cracker{
		ctx:             ctx,
		newUniqueHashes: make(map[string]struct{}),
	}

	prog, err := pattern.Compile("pattern.txt", patternSrc, pattern.CompileOptions{})
	if err != nil {
		return nil, err
	}
	cli.Print("Pattern compiled successfully (total complexity: %d ≈ %.2e, max length: %d)", prog.Comp, float64(prog.Comp), prog.MaxLen())

	var workerErr error
	done := make(chan error)
	go func() {
		done <- crack(c, prog, targetHashes)
		close(done)
	}()

loop:
	for range time.Tick(500 * time.Millisecond) {
		// update CLI status
		c.mu.Lock()
		tries := c.triesCnt
		lastStr := c.lastStr
		c.mu.Unlock()

		rateStr := "???"
		if c.triesPerSecondBuf.Len() != 0 {
			// Average the buffer values
			a, b := c.triesPerSecondBuf.PeekAll()
			rate := (util.Sum(a) + util.Sum(b)) / float64(c.triesPerSecondBuf.Len())
			rateStr = fmt.Sprintf("%.2fMH/s", rate/1e6)
		}
		extraStatus := c.extraStatusStr
		if extraStatus != "" {
			extraStatus = ", " + extraStatus
		}
		cli.Status("Progress=%.3f%%, Rate=%s, Last=%q%s", float64(tries)/float64(prog.Comp)*100, rateStr, lastStr, extraStatus)

		for c.triesPerSecondBuf.Len() > 20 {
			c.triesPerSecondBuf.Read()
		}

		c.mu.Lock()
		for _, s := range c.msgBuf {
			cli.Print("%s", s)
		}
		c.msgBuf = c.msgBuf[:0]
		c.mu.Unlock()

		select {
		case workerErr = <-done:
			if workerErr == nil {
				cli.Print("Worker done")
			}
			break loop
		default:
		}
		if ctx.Err() != nil {
			cli.Print("Shutting down worker")
			workerErr = <-done
			break loop
		}
	}

	if err := workerErr; err != nil {
		return nil, err
	}
	return c.newHashes, nil
}

func (c *cracker) Msg(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	c.mu.Lock()
	c.msgBuf = append(c.msgBuf, s)
	c.mu.Unlock()
}

func (c *cracker) Status(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	c.mu.Lock()
	c.extraStatusStr = s
	c.mu.Unlock()
}

func crack(c *cracker, prog pattern.Segment, targetHashes []uint64) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	platforms, err := cl.GetPlatformIDs()
	if err != nil {
		return err
	}
	if len(platforms) == 0 {
		return fmt.Errorf("no OpenCL platforms")
	}
	platform := platforms[0]
	var platformName string
	if err := cl.GetPlatformInfo(platform, cl.PLATFORM_NAME, &platformName); err != nil {
		return err
	}
	devices, err := cl.GetDeviceIDs(platform, cl.DEVICE_TYPE_GPU)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return fmt.Errorf("no OpenCL devices")
	}
	device := devices[0]
	var deviceName string
	if err := cl.GetDeviceInfo(device, cl.DEVICE_NAME, &deviceName); err != nil {
		return err
	}
	c.Msg("Using OpenCL platform %q with device %q", platformName, deviceName)

	crackerOpts := pcl.Options{
		Workers: 256,
		Tries:   8192,
	}
	tuner := NewTuner(crackerOpts.Workers, crackerOpts.Tries)

	c.Msg("Initializing buffers and compiling OpenCL kernel")
	cr, err := pcl.NewCracker(device, prog, targetHashes, crackerOpts)
	if err != nil {
		return err
	}
	defer cr.Delete()

	//os.WriteFile("program.cl", []byte(cr.DebugInfo.OpenClCode), 0666)

	c.Msg("Making guesses")
	c.Status("Warming up / collecting baseline")
	prevTotalIdx := 0
	var prevTime time.Time
	idx := prog.MakeIndex()
	for {
		matches, err := cr.Dispatch()
		if err == pcl.Done {
			break
		} else if err != nil {
			return err
		}
		idx.Reset()
		if cr.TotalIdx() > 0 {
			idx.Add(prog, cr.TotalIdx())
		}

		for _, s := range matches {
			c.Msg("found: %016x = %s", hash.Murmur64aSum(s), s)
		}

		now := time.Now()
		newTries := cr.TotalIdx() - prevTotalIdx
		var lastStr string
		{
			idx.Reset()
			idx.Add(prog, cr.TotalIdx())
			lastStr = prog.StringAt(idx)
		}
		triesPerSecond := float64(-1)
		if !prevTime.IsZero() {
			triesPerSecond = float64(newTries) / now.Sub(prevTime).Seconds()
		}

		c.mu.Lock()
		c.newHashes = append(c.newHashes, matches...)
		for _, h := range matches {
			c.newUniqueHashes[h] = struct{}{}
		}
		c.triesCnt += newTries
		c.lastStr = lastStr
		if triesPerSecond >= 0 {
			c.triesPerSecondBuf.Write(triesPerSecond)
		}
		allFound := len(c.newUniqueHashes) == len(targetHashes)
		c.mu.Unlock()

		if w, t, done, changed := tuner.Step(int(cr.LastKernelRunDuration().Nanoseconds()), newTries); changed {
			var tuneStr string
			if done {
				tuneStr = "Tuned"
			} else {
				tuneStr = "Tuning"
			}
			c.Status("%s (WxT=%dx%d)", tuneStr, w, t)
			cr.ChangeNumWorkers(w)
			cr.ChangeNumTries(t)
		}

		prevTotalIdx = cr.TotalIdx()
		prevTime = now

		if allFound {
			c.Msg("All hashes found")
			break
		}

		if c.ctx.Err() != nil {
			return err
		}
	}
	return nil
}

func run() error {
	var epilog strings.Builder
	{
		// TODO: Pattern syntax guide
	}

	argp := argparse.NewParser("hd2-hash-cracker", "Helldivers 2 filename hash cracking tool", &argparse.ParserConfig{
		EpiLog: epilog.String(),
	})
	optExpr := argp.Flag("e", "expr", &argparse.Option{
		Help: "evaluate expression instead of input file",
	})
	optInput := argp.String("", "input", &argparse.Option{
		Help:       "input file (or expression if -e is set)",
		Positional: true,
		Required:   true,
	})
	optHashes := argp.String("t", "target", &argparse.Option{
		Help:     "file listing target hashes to crack",
		Default:  "hd2-patterns/target.txt",
		Required: true,
	})
	optOutput := argp.String("o", "output", &argparse.Option{
		Help:    "output file to append found hashes to",
		Default: "cracked.txt",
	})
	optCpuProfile := argp.Flag("", "cpuprofile", &argparse.Option{
		Help:      "write CPU profile",
		HideEntry: true,
	})

	if err := argp.Parse(nil); err != nil {
		if errors.Is(err, argparse.BreakAfterHelpError) {
			return nil
		}
		return err
	}

	if *optCpuProfile {
		const filename = "cpu.prof"
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("creating CPU profile file: %w", err)
		}
		cli.Print("Starting CPU profile")
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting CPU profile: %w", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			cli.Print("CPU profile written to %s", filename)
		}()
	}

	var patternSrc []byte
	if *optExpr {
		patternSrc = []byte(*optInput)
	} else {
		b, err := os.ReadFile(*optInput)
		if err != nil {
			return fmt.Errorf("reading input file: %w", err)
		}
		patternSrc = b
	}

	var targetHashes []uint64
	{
		b, err := os.ReadFile(*optHashes)
		if err != nil {
			return fmt.Errorf("reading target hashes file: %w", err)
		}
		for line := range bytes.SplitSeq(b, []byte("\n")) {
			line = bytes.TrimSuffix(line, []byte("\r"))
			if len(line) == 0 {
				continue
			}

			h, err := hash.Parse64(string(line))
			if err != nil {
				return fmt.Errorf("parsing target hash: %w", err)
			}
			targetHashes = append(targetHashes, h)
		}
		slices.Sort(targetHashes)
		util.Uniq(targetHashes)
	}

	cli.Print("Ctrl+C to quit")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	newHashes, err := runCracker(ctx, patternSrc, targetHashes)
	if err != nil {
		return err
	}

	// Write back new hash strings to output file by appending and deduplicating
	{
		cli.Print("Adding %d hashes to %s", len(newHashes), *optOutput)
		b, err := os.ReadFile(*optOutput)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		var lines [][]byte
		if b != nil {
			lines = bytes.Split(b, []byte("\n"))
			for i := range lines {
				lines[i] = bytes.TrimSuffix(lines[i], []byte("\r"))
			}
			lines = slices.DeleteFunc(lines, func(b []byte) bool { return len(b) == 0 })
		}
		for _, h := range newHashes {
			lines = append(lines, []byte(h))
		}
		slices.SortFunc(lines, bytes.Compare)
		util.UniqFunc(lines, bytes.Equal)
		if err := os.WriteFile(*optOutput, bytes.Join(lines, []byte("\n")), 0666); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		cli.Error("Error: %v", err)
	}
	os.Stderr.WriteString("\n")
}
