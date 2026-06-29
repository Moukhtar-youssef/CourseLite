// package utils provide functions and utils that help in stream lining programming
package utils

import "strings"

func SanitizeLog(input string) string {
	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, "\r", "")
	return input
}
