package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/engine"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	"github.com/parsaazari/fuzznexus/pkg/input"
	"github.com/parsaazari/fuzznexus/pkg/interactive"
	"github.com/parsaazari/fuzznexus/pkg/runner"
)

// flagRegistry describes the registered flags for the segmented help output
// (see Usage in help.go). It is (re)populated on every ParseFlags call.
var flagRegistry *fuzznexus.FlagRegistry

// ParseFlags registers every CLI flag from fuzznexus.RegisterFlags — the single source
// of truth — onto the global flag set and parses the command line into opts.
// Flag defaults are the current field values, so a value loaded from a config
// file is used unless the same flag is given on the command line (file < CLI).
func ParseFlags(opts *fuzznexus.ConfigOptions) *fuzznexus.ConfigOptions {
	flagRegistry = fuzznexus.RegisterFlags(flag.CommandLine, opts)
	flag.Usage = Usage
	flag.Parse()
	return opts
}

func main() {

	var err, optserr error
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// prepare the default config options from default config file
	var opts *fuzznexus.ConfigOptions
	opts, optserr = fuzznexus.ReadDefaultConfig()

	opts = ParseFlags(opts)

	// Handle searchhash functionality and exit
	if opts.General.Searchhash != "" {
		coptions, pos, err := engine.SearchHash(opts.General.Searchhash)
		if err != nil {
			fmt.Printf("[ERR] %s\n", err)
			os.Exit(1)
		}
		if len(coptions) > 0 {
			fmt.Printf("Request candidate(s) for hash %s\n", opts.General.Searchhash)
		}
		for _, copt := range coptions {
			conf, err := fuzznexus.ConfigFromOptions(&copt.ConfigOptions, ctx, cancel)
			if err != nil {
				continue
			}
			ok, reason := engine.HistoryReplayable(conf)
			if ok {
				printSearchResults(conf, pos, copt.Time, opts.General.Searchhash)
			} else {
				fmt.Printf("[ERR] Hash cannot be mapped back because %s\n", reason)
			}

		}
		if err != nil {
			fmt.Printf("[ERR] %s\n", err)
		}
		os.Exit(0)
	}

	if opts.General.ShowVersion {
		fmt.Printf("FuzzNexus version: %s\n", fuzznexus.Version())
		os.Exit(0)
	}
	if len(opts.Output.DebugLog) != 0 {
		f, err := os.OpenFile(opts.Output.DebugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Disabling logging, encountered error(s): %s\n", err)
			log.SetOutput(io.Discard)
		} else {
			log.SetOutput(f)
			defer f.Close()
		}
	} else {
		log.SetOutput(io.Discard)
	}
	if optserr != nil {
		log.Printf("Error while opening default config file: %s", optserr)
	}

	if opts.General.ConfigFile != "" {
		opts, err = fuzznexus.ReadConfig(opts.General.ConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Encoutered error(s): %s\n", err)
			Usage()
			fmt.Fprintf(os.Stderr, "Encoutered error(s): %s\n", err)
			os.Exit(1)
		}
		// Reset the flag package state
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		// Re-parse the cli options
		opts = ParseFlags(opts)
	}

	// Set up Config struct
	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		Usage()
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		os.Exit(1)
	}

	job, err := assembly.BuildJob(conf)

	if job.AuditLogger != nil {
		defer job.AuditLogger.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		Usage()
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		os.Exit(1)
	}
	if err := SetupFilters(opts, conf); err != nil {
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		Usage()
		fmt.Fprintf(os.Stderr, "Encountered error(s): %s\n", err)
		os.Exit(1)
	}

	if !conf.Noninteractive {
		go func() {
			err := interactive.Handle(job)
			if err != nil {
				log.Printf("Error while trying to initialize interactive session: %s", err)
			}
		}()
	}

	// Job handles waiting for goroutines to complete itself
	job.Start()
}

// SetupFilters installs the matchers and filters onto conf.MatcherManager. The
// matcher/filter construction itself lives in filter.FromConfig (pure, testable);
// this shim owns only the one decision that genuinely needs the global CLI flag
// set - whether to install the default status matcher - plus the -ignore-body
// warning.
func SetupFilters(parseOpts *fuzznexus.ConfigOptions, conf *fuzznexus.Config) error {
	// The default status matcher is installed unless another matcher was explicitly
	// passed on the CLI. Only the global flag set knows which flags were passed vs
	// defaulted, so this decision cannot move into the library.
	statusSet := false
	matcherSet := false
	// responseMatcherSet tracks the response-body matchers (ms/ml/mw) explicitly
	// passed on the CLI. The -ignore-body warning keys off CLI-passed flags (as the
	// original did via flag.Visit), NOT off the config value, so a config-file /
	// .fuzznexusrc value does not spuriously trigger it.
	responseMatcherSet := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "mc":
			statusSet = true
		case "ms", "ml", "mw":
			matcherSet = true
			responseMatcherSet = true
		case "mr", "mt":
			matcherSet = true
		}
	})

	mm, err := filter.FromConfig(parseOpts, statusSet || !matcherSet)
	conf.MatcherManager = mm

	warningIgnoreBody := responseMatcherSet ||
		parseOpts.Filter.Size != "" || parseOpts.Filter.Lines != "" || parseOpts.Filter.Words != ""
	if conf.IgnoreBody && warningIgnoreBody {
		fmt.Printf("*** Warning: possible undesired combination of -ignore-body and the response options: fl,fs,fw,ml,ms and mw.\n")
	}
	return err
}

func printSearchResults(conf *fuzznexus.Config, pos int, exectime time.Time, hash string) {
	inp, err := input.NewInputProvider(conf)
	if err.ErrorOrNil() != nil {
		fmt.Printf("-------------------------------------------\n")
		fmt.Println("Encountered error that prevents reproduction of the request:")
		fmt.Println(err.ErrorOrNil())
		return
	}
	inp.SetPosition(pos)
	inputdata := inp.Value()
	inputdata["FNHASH"] = []byte(hash)
	basereq := fuzznexus.BaseRequest(conf)
	dummyrunner := runner.NewSimpleRunner(conf, false)
	ffufreq, _ := dummyrunner.Prepare(inputdata, &basereq)
	rawreq, _ := dummyrunner.Dump(&ffufreq)
	fmt.Printf("-------------------------------------------\n")
	fmt.Printf("FuzzNexus job started at: %s\n\n", exectime.Format(time.RFC3339))
	fmt.Printf("%s\n", string(rawreq))
}
