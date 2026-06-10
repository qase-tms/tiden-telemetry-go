package tiden

import (
	"runtime"
	"strings"
)

// newStacktrace captures the calling goroutine's stack. skip is the number of
// frames to omit on top of this function (so the first reported frame is the
// caller's caller). Frames are returned outermost-first with the capture/crash
// site last — the order the Tiden UI expects.
func newStacktrace(skip int) *Stacktrace {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return nil
	}

	it := runtime.CallersFrames(pcs[:n])
	var frames []Frame
	for {
		f, more := it.Next()
		if f.Function != "" || f.File != "" {
			frames = append(frames, Frame{
				Function: f.Function,
				Filename: shortFile(f.Function, f.File),
				AbsPath:  f.File,
				Lineno:   f.Line,
				InApp:    inApp(f.File, f.Function),
			})
		}
		if !more {
			break
		}
	}
	if len(frames) == 0 {
		return nil
	}

	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &Stacktrace{Frames: frames}
}

// inApp marks a frame as application code (vs the standard library, the module
// cache, or this SDK).
func inApp(file, fn string) bool {
	if file == "" {
		return false
	}
	if strings.Contains(file, "/pkg/mod/") {
		return false
	}
	if strings.HasPrefix(fn, "runtime.") || strings.HasPrefix(fn, modulePrefix) {
		return false
	}
	return true
}

// shortFile derives a stable "pkgpath/file.go" name from the fully-qualified
// function name and the absolute file path, so filenames don't leak the build
// machine's directory layout (abs_path keeps the full path).
func shortFile(fn, file string) string {
	base := file
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	pkg := fn
	if i := strings.LastIndexByte(pkg, '/'); i >= 0 {
		dir, leaf := pkg[:i+1], pkg[i+1:]
		if d := strings.IndexByte(leaf, '.'); d >= 0 {
			leaf = leaf[:d]
		}
		pkg = dir + leaf
	} else if d := strings.IndexByte(pkg, '.'); d >= 0 {
		pkg = pkg[:d]
	}
	if pkg == "" || base == "" {
		return base
	}
	return pkg + "/" + base
}
