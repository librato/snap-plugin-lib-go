package runner

import (
	"flag"
	"fmt"
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
)

type logLevelHandler struct {
	opt *plugin.Options
}

func (l *logLevelHandler) String() string {
	if l.opt == nil {
		return "error"
	}

	return l.opt.LogLevel.String()
}

func (l *logLevelHandler) Set(s string) error {
	// accept level as a form of int (0 - 6)
	intLvl, errConv := strconv.Atoi(s)
	if errConv == nil && intLvl >= int(logrus.PanicLevel) && intLvl <= int(logrus.TraceLevel) {
		l.opt.LogLevel = logrus.Level(intLvl)
		return nil
	}

	// accept level as a form os string (warning, error etc.)
	lvl, errParse := logrus.ParseLevel(s)
	if errParse != nil {
		return errParse
	}
	l.opt.LogLevel = lvl

	return nil
}

func newFlagParser(name string, opt *plugin.Options) *flag.FlagSet {
	flagParser := flag.NewFlagSet(name, flag.ContinueOnError)

	flagParser.StringVar(&opt.GrpcIp,
		"grpc-ip", "127.0.0.1",
		"IP Address on which GRPC server will be served")

	flagParser.IntVar(&opt.GrpcPort,
		"grpc-port", 0,
		"Port on which GRPC server will be served")

	allLogLevels := strings.Replace(fmt.Sprintf("%v", logrus.AllLevels), " ", ", ", -1)
	flagParser.Var(&logLevelHandler{opt: opt},
		"log-level",
		fmt.Sprintf("Minimal level of logged messages %s", allLogLevels))

	flagParser.BoolVar(&opt.EnablePprof,
		"enable-pprof", false,
		"Enable profiling server")

	flagParser.BoolVar(&opt.EnableStats,
		"enable-stats", false,
		"Enable stats server")

	flagParser.BoolVar(&opt.DebugMode,
		"debug-mode", false,
		"Run plugin in debug mode (standalone)")

	flagParser.StringVar(&opt.PluginConfig,
		"plugin-config", "{}",
		"Collector configuration in debug mode")

	return flagParser
}

func ParseCmdLineOptions(pluginName string, args []string) (*plugin.Options, error) {
	opt := &plugin.Options{
		LogLevel: logrus.WarnLevel, // todo: should i leave it here?
	}

	flagParser := newFlagParser(pluginName, opt)

	err := flagParser.Parse(args)
	if err != nil {
		return opt, fmt.Errorf("can't parse command line options: %v", err)
	}

	v := flagParser.Args()
	if len(v) > 0 {
		return opt, fmt.Errorf("unexpected option(s) provided: %v %v", v, len(v))
	}

	return opt, nil
}

func ValidateOptions(opt *plugin.Options) error {
	grpcIp := net.ParseIP(opt.GrpcIp)
	if grpcIp == nil {
		return fmt.Errorf("GRPC IP contains invalid address")
	}

	return nil
}
