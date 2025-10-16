//nolint:revive
package common

func ConvertLineToActionLine(line int) int {
	if line < 1 {
		return 1
	}
	return line
}
