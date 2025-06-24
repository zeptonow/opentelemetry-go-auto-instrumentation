// Copyright (c) 2024 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/config"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/errc"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/instrument"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/preprocess"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/util"
)

const (
	SubcommandSet     = "set"
	SubcommandGo      = "go"
	SubcommandVersion = "version"
	SubcommandRemix   = "remix"
)

var usage = `Usage: {} <command> [args]
Example:
	{} go build
	{} go install
	{} go build main.go
	{} version
	{} set -verbose -rule=custom.json

Command:
	version    print the version
	set        set the configuration
	go         build the Go application
`

func printUsage() {
	name, _ := util.GetToolName()
	usage = strings.ReplaceAll(usage, "{}", name)
	fmt.Print(usage)
}

func initTempDir() error {
	// All temp directories are prepared before, instrument phase should not
	// create any new directories.
	if util.GetRunPhase() == util.PInstrument {
		return nil
	}

	// Make temp build directory if not exists
	if util.PathNotExists(util.TempBuildDir) {
		err := os.MkdirAll(util.TempBuildDir, 0777)
		if err != nil {
			return errc.New(err.Error())
		}
	}
	// Recreate preprocess/instrument subdirectories if they already exist
	for _, subdir := range []string{util.PPreprocess, util.PInstrument} {
		_ = os.RemoveAll(util.GetTempBuildDirWith(subdir))
		_ = os.MkdirAll(util.GetTempBuildDirWith(subdir), 0777)
	}

	return nil
}

func initEnv() error {
	util.Assert(len(os.Args) >= 2, "no command specified")

	// Determine the run phase
	switch {
	case strings.HasSuffix(os.Args[1], SubcommandGo):
		// otel go build?
		util.SetRunPhase(util.PPreprocess)
	case os.Args[1] == SubcommandRemix:
		// otel remix?
		util.SetRunPhase(util.PInstrument)
	default:
		// do nothing
	}

	// Create temp build directory
	err := initTempDir()
	if err != nil {
		return err
	}

	// Prepare shared configuration
	if util.InPreprocess() || util.InInstrument() {
		err = config.InitConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func fatal(err error) {
	message := "===== Environments =====\n"
	message += fmt.Sprintf("%-11s: %s\n", "command", strings.Join(os.Args, " "))
	message += fmt.Sprintf("%-11s: %s\n", "errorLog", util.GetLoggerPath())
	message += fmt.Sprintf("%-11s: %s\n", "workDir", os.Getenv("PWD"))
	message += fmt.Sprintf("%-11s: %s, %s, %s\n", "toolchain",
		runtime.GOOS+"/"+runtime.GOARCH,
		runtime.Version(), config.ToolVersion)
	if perr, ok := err.(*errc.PlentifulError); ok {
		if len(perr.Details) > 0 {
			for k, v := range perr.Details {
				message += fmt.Sprintf("%-11s: %s\n", k, v)
			}
		}
	}
	message += "\n===== Fatal Error ======\n"
	if perr, ok := err.(*errc.PlentifulError); ok {
		message += "\n" + perr.Reason
	}
	util.LogFatal("%s", message) // log in red color
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	err := initEnv()
	if err != nil {
		fatal(err)
	}

	subcmd := os.Args[1]
	switch subcmd {
	case SubcommandVersion:
		err = config.PrintVersion()
	case SubcommandSet:
		err = config.Configure()
	case SubcommandGo:
		err = preprocess.Preprocess()
	case SubcommandRemix:
		err = instrument.Instrument()
		if err != nil {
			// We do not want to print the usage message in remix phase, because
			// its caller(preprocess) phase has already collected the error msg
			// and handle it properly.
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
	default:
		printUsage()
	}
	if err != nil {
		fatal(err)
	}
}
