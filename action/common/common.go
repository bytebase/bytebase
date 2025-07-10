package common

func ConvertLineToActionLine(line int) int {
	if line < 0 {
		return 1
	}
	return line + 1
}
