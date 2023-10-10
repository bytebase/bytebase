package stacktrace

import (
	"runtime"
	"strconv"
)

// TakeStacktrace takes at most n stacks, skiping the first skip stacks.
func TakeStacktrace(n, skip uint) []byte {
	var buf []byte
	pcs := make([]uintptr, n)

	// +2 to exclude runtime.Callers and TakeStacktrace
	numFrames := runtime.Callers(2+int(skip), pcs)
	if numFrames == 0 {
		return buf
	}
	frames := runtime.CallersFrames(pcs[:numFrames])
	for i := 0; ; i++ {
		frame, more := frames.Next()
		if i != 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, []byte(frame.Function)...)
		buf = append(buf, '\n')
		buf = append(buf, '\t')
		buf = append(buf, []byte(frame.File)...)
		buf = append(buf, ':')
		buf = append(buf, []byte(strconv.Itoa(frame.Line))...)
		if !more {
			break
		}
	}
	return buf
}
