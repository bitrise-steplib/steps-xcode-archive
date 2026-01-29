package stringutil

import (
	"bufio"
	"math"
	"strings"
)

// ReadFirstLine returns the first line from the input string.
// If isIgnoreLeadingEmptyLines is true, it skips empty lines and returns the first non-empty line.
func ReadFirstLine(s string, isIgnoreLeadingEmptyLines bool) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	firstLine := ""
	for scanner.Scan() {
		firstLine = scanner.Text()
		if !isIgnoreLeadingEmptyLines || firstLine != "" {
			break
		}
	}
	return firstLine
}

// MaxLastCharsWithDots returns the last maxCharCount characters from inStr.
// If the string is longer than maxCharCount, it prepends "..." to indicate truncation.
// Returns empty string if maxCharCount < 4.
func MaxLastCharsWithDots(inStr string, maxCharCount int) string {
	return genericTrim(inStr, maxCharCount, true, true)
}

// MaxFirstCharsWithDots returns the first maxCharCount characters from inStr.
// If the string is longer than maxCharCount, it appends "..." to indicate truncation.
// Returns empty string if maxCharCount < 4.
func MaxFirstCharsWithDots(inStr string, maxCharCount int) string {
	return genericTrim(inStr, maxCharCount, false, true)
}

// LastNLines returns the last n lines from the input string.
// It trims leading and trailing newlines before splitting.
// If the string has fewer than n lines, returns all lines.
func LastNLines(s string, n int) string {
	trimmed := strings.Trim(s, "\n")
	splitted := strings.Split(trimmed, "\n")

	if len(splitted) >= n {
		splitted = splitted[len(splitted)-n:]
	}

	return strings.Join(splitted, "\n")
}

// IndentTextWithMaxLength formats text with indentation and wraps lines longer than maxTextLineCharWidth.
// Each line is prefixed with indent string.
// If isIndentFirstLine is false, the first line will not be indented.
func IndentTextWithMaxLength(text, indent string, maxTextLineCharWidth int, isIndentFirstLine bool) string {
	if maxTextLineCharWidth < 1 {
		return ""
	}

	formattedText := ""

	addLine := func(line string) {
		isFirstLine := (formattedText == "")
		if isFirstLine && !isIndentFirstLine {
			formattedText = line
		} else {
			if !isFirstLine {
				formattedText += "\n"
			}
			formattedText += indent + line
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		lineLength := len(line)
		if lineLength > maxTextLineCharWidth {
			lineCnt := math.Ceil(float64(lineLength) / float64(maxTextLineCharWidth))
			for i := 0; i < int(lineCnt); i++ {
				startIdx := i * maxTextLineCharWidth
				endIdx := startIdx + maxTextLineCharWidth
				if endIdx > lineLength {
					endIdx = lineLength
				}
				addLine(line[startIdx:endIdx])
			}
		} else {
			addLine(line)
		}
	}

	return formattedText
}

func genericTrim(inStr string, maxCharCount int, trimmAtStart, appendDots bool) string {
	strLen := len(inStr)

	if maxCharCount >= strLen {
		return inStr
	}

	if appendDots && maxCharCount < 4 {
		return ""
	}

	var retStr string
	if trimmAtStart {
		if appendDots {
			retStr = inStr[strLen-(maxCharCount-3):]
			retStr = "..." + retStr
		} else {
			retStr = inStr[strLen-maxCharCount:]
		}
	} else {
		if appendDots {
			retStr = inStr[:maxCharCount-3]
			retStr = retStr + "..."
		} else {
			retStr = inStr[:maxCharCount]
		}
	}

	return retStr
}
