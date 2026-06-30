package v1

import colorpb "google.golang.org/genproto/googleapis/type/color"

func isOpaqueColor(c *colorpb.Color) bool {
	if c == nil {
		return false
	}
	if !isUnitFloat(c.Red) || !isUnitFloat(c.Green) || !isUnitFloat(c.Blue) {
		return false
	}
	if c.Alpha != nil && c.Alpha.Value != 1 {
		return false
	}
	return true
}

func isUnitFloat(v float32) bool {
	return v >= 0 && v <= 1
}
