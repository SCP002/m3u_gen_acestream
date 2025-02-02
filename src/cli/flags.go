package cli

import (
	"github.com/cockroachdb/errors"
	goFlags "github.com/jessevdk/go-flags"
	pLog "github.com/phuslu/log"
)

// Flags represents command line flags.
type Flags struct {
	Version  bool       `short:"v" long:"version" description:"Print the program version"`
	LogLevel pLog.Level `short:"l" long:"logLevel" description:"Logging level. Can be from 1 (most verbose) to 7 (least verbose)"`
	LogFile  string     `short:"f" long:"logFile" description:"Log file. If set, writes structured log to a file at the specified path"`
	CfgPath  string     `short:"c" long:"cfgPath" description:"Config file path to read from or initialize a default"`
}

// Parse returns a structure initialized with command line arguments and error if parsing failed.
func Parse() (Flags, error) {
	flags := Flags{
		// Set defaults
		LogLevel: pLog.InfoLevel,
		CfgPath:  "m3u_gen_acestream.yaml",
	}
	parser := goFlags.NewParser(&flags, goFlags.Options(goFlags.Default))
	_, err := parser.Parse()
	return flags, errors.Wrap(err, "Parse command line flags")
}

// IsErrOfType returns true if `err` is of type `t`.
func IsErrOfType(err error, t goFlags.ErrorType) bool {
	goFlagsErr := &goFlags.Error{}
	if ok := errors.As(err, &goFlagsErr); ok && goFlagsErr.Type == t {
		return true
	}
	return false
}
