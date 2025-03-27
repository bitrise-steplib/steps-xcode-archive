package step

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/exportoptions"
)

func generateAdditionalOptions(platform string, customOptions []string) []string {
	destination := "generic/platform=" + platform
	destinationOptions := []string{"-destination", destination}

	var options []string
	if len(customOptions) != 0 {
		if !sliceutil.IsStringInSlice("-destination", customOptions) {
			options = append(options, destinationOptions...)
		}

		options = append(options, customOptions...)
	} else {
		options = append(options, destinationOptions...)
	}

	return options
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

func printLastLinesOfXcodebuildLog(logger log.Logger, xcodebuildLog string, isXcodebuildSuccess bool) {
	const lastLinesMsg = "\nLast lines of the Xcode log:"
	if isXcodebuildSuccess {
		logger.Infof(lastLinesMsg)
	} else {
		logger.Infof(colorstring.Red(lastLinesMsg))
	}

	logger.Printf("%s", stringutil.LastNLines(xcodebuildLog, 20))
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
