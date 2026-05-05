package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/stringmerge"
)

func AddContentOrCreateFile(
	logger log.Logger,
	osProxy OsProxy,
	filePath string,
	blockSuffix string,
	content string,
) error {
	// Check if the file exists
	currentContent, exists, err := osProxy.ReadFileIfExists(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	if !exists {
		logger.Debugf("File %s does not exist, creating", filePath)
	}

	content = stringmerge.ChangeContentInBlock(
		currentContent,
		fmt.Sprintf("# [start] %s", strings.TrimSpace(blockSuffix)),
		fmt.Sprintf("# [end] %s", strings.TrimSpace(blockSuffix)),
		content,
	)

	err = osProxy.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	logger.Debugf("Updated file %s with content in block %s", filePath, blockSuffix)

	return nil
}
