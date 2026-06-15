package vips_bridge

/*
#cgo pkg-config: vips
#include "vips_save.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/davidbyttow/govips/v2/vips"
)

// libvips uses a global error buffer, so we use a mutex to make sure
// concurrent saves don't overwrite each other's error messages.
var vipsMu sync.Mutex

// SaveToVipsFile writes the image directly to disk. This is much more memory
// efficient and handles some AVIF builds where buffer exports are broken.
func SaveToVipsFile(img *vips.ImageRef, tmpPath string, quality int) error {
	vipsPath := fmt.Sprintf("%s[Q=%d]", tmpPath, quality)
	cPath := C.CString(vipsPath)
	defer C.free(unsafe.Pointer(cPath))

	// We have to use reflection to get the underlying C pointer
	// since the govips package doesn't export it.
	imgVal := reflect.ValueOf(img).Elem()
	imgPtrField := imgVal.FieldByName("image")
	if !imgPtrField.IsValid() {
		return fmt.Errorf("could not access internal vips image pointer")
	}

	imgPtr := (*C.VipsImage)(unsafe.Pointer(imgPtrField.Pointer()))

	vipsMu.Lock()
	ret := C.vips_save_to_file(imgPtr, cPath)

	var msg string
	if ret != 0 {
		cErr := C.vips_get_last_error()
		if cErr != nil {
			msg = C.GoString(cErr)
			C.free(unsafe.Pointer(cErr))
		}
	}
	vipsMu.Unlock()

	// Make sure the Go GC doesn't collect 'img'
	// while we're still running that C function.
	runtime.KeepAlive(img)

	if ret != 0 {
		if msg == "" {
			msg = "unknown libvips error"
		}
		return fmt.Errorf("libvips error: %s", msg)
	}

	return nil
}

// HasLoader returns true if libvips has a loader for the given format nickname.
func HasLoader(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return int(C.vips_has_loader(cName)) != 0
}

// HasSaver returns true if libvips has a saver for the given format nickname.
func HasSaver(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return int(C.vips_has_saver(cName)) != 0
}
