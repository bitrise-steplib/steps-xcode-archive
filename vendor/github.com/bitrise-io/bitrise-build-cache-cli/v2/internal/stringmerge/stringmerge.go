package stringmerge

import "strings"

// ChangeContentInBlock - checks the currentContent whether a `blockStartPattern` and `blockEndPattern` block is already present.
// If there is, then only the block's content will be modified.
// If there's no marked block in the content yet then append it to the existing content
// with the `blockContentStr` content in the block.
func ChangeContentInBlock(currentContent, blockStartPattern, blockEndPattern, blockContentStr string) string {
	fullBlockContent := blockStartPattern + "\n" + blockContentStr + "\n" + blockEndPattern + "\n"

	// if current content is empty then just return the block content
	if len(currentContent) < 1 {
		return fullBlockContent
	}

	// check if the block is already present
	startIndex := strings.Index(currentContent, blockStartPattern)
	endIndex := strings.Index(currentContent, blockEndPattern)

	if startIndex > -1 && endIndex > -1 && startIndex < endIndex {
		// the block is already present, only replace the content
		return currentContent[:startIndex] +
			blockStartPattern + "\n" +
			blockContentStr + "\n" +
			currentContent[endIndex:]
	}

	// the block is not present yet, append it to the existing content
	return currentContent + "\n" + fullBlockContent
}
