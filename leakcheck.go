package main

// #cgo CPPFLAGS: -fsanitize=address
// #cgo LDFLAGS: -fsanitize=address
//
// #include <sanitizer/lsan_interface.h>
import "C"

import "runtime"

// Call LLVM Leak Sanitizer's at-exit hook that doesn't
// get called automatically by Go.
func DoLeakSanitizerCheck() {
        runtime.GC()
        C.__lsan_do_leak_check()
}