package step

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/log/colorstring"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
)

// filterSPMAdditionalOptions narrows the xcodebuild_options input to what -showBuildSettings
// accepts: the Swift package flags and build setting overrides.
func filterSPMAdditionalOptions(xcodebuildAdditionalOptions []string) []string {
	spmFlags := map[string]bool{
		"-skipPackagePluginValidation":            true,
		"-skipMacroValidation":                    true,
		"-skipPackageUpdates":                     true,
		"-disableAutomaticPackageResolution":      true,
		"-onlyUsePackageVersionsFromResolvedFile": true,
		"-clonedSourcePackagesDirPath":            true,
	}

	options, _ := xcodecommand.ParseAdditionalOptions(xcodebuildAdditionalOptions)
	filtered := options.Filter(func(o xcodecommand.Option) bool {
		return o.Kind == xcodecommand.BuildSetting || spmFlags[o.Name]
	}).Args()
	if filtered == nil {
		return []string{}
	}

	return filtered
}

func determineExportMethod(desiredExportMethod string, archiveExportMethod exportoptions.Method, logger log.Logger) (exportoptions.Method, error) {
	if desiredExportMethod == "auto-detect" {
		logger.Printf("auto-detect export method specified: using the archive profile's export method: %s", archiveExportMethod)
		return archiveExportMethod, nil
	}

	exportMethod, err := exportoptions.ParseMethod(desiredExportMethod)
	if err != nil {
		return "", fmt.Errorf("failed to parse export method: %s", err)
	}
	logger.Printf("export method specified: %s", desiredExportMethod)

	return exportMethod, nil
}

func printLastLinesOfXcodebuildLog(logger log.Logger, fileManager fileutil.FileManager, xcodebuildLog string, isXcodebuildSuccess bool) {
	const lastLinesMsg = "\nLast lines of the Xcode log:"
	if isXcodebuildSuccess {
		logger.Infof(lastLinesMsg)
	} else {
		logger.Infof(colorstring.Red(lastLinesMsg))
	}

	logger.Printf("%s", fileManager.LastNLines(xcodebuildLog, 20))
	logger.Println()

	if !isXcodebuildSuccess {
		logger.Warnf("If you can't find the reason of the error in the log, please check the artifact %s.", xcodebuildArchiveLogFilename)
	}

	logger.Infof(colorstring.Magenta(fmt.Sprintf(`
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $%s environment variable.

Deploy to Bitrise.io Step can attach the file to your build as an artifact.`, xcodebuildArchiveLogPathEnvKey)))
}

func findIDEDistrubutionLogsPath(output string, logger log.Logger) (string, error) {
	pattern := `IDEDistribution: -\[IDEDistributionLogging _createLoggingBundleAtPath:\]: Created bundle at path ['\"](?P<log_path>.*)['\"]`
	re := regexp.MustCompile(pattern)

	logger.Printf("Locating IDE distrubution logs path")

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); len(match) == 2 {
			logger.Printf("Located IDE distrubution logs path")

			return match[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	logger.Printf("IDE distrubution logs path not found")

	return "", nil
}
