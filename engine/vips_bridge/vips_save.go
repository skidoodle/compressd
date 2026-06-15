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
	"unsafe"

	"github.com/davidbyttow/govips/v2/vips"
)

// SaveToVipsFile writes an image directly to disk using libvips's native savers.
// This bypasses the Go heap and avoids issues with missing buffer-based savers.
func SaveToVipsFile(img *vips.ImageRef, tmpPath string, quality int) error {
	// The [Q=quality] suffix is a libvips convention to pass options to the saver.
	vipsPath := fmt.Sprintf("%s[Q=%d]", tmpPath, quality)
	cPath := C.CString(vipsPath)
	defer C.free(unsafe.Pointer(cPath))

	// We use reflection to grab the unexported C pointer from the govips ImageRef.
	imgVal := reflect.ValueOf(img).Elem()
	imgPtrField := imgVal.FieldByName("image")
	if !imgPtrField.IsValid() {
		return fmt.Errorf("could not access internal vips image pointer")
	}

	imgPtr := (*C.VipsImage)(unsafe.Pointer(imgPtrField.Pointer()))

	if err := C.vips_save_to_file(imgPtr, cPath); err != 0 {
		cErr := C.vips_get_last_error()
		defer C.free(unsafe.Pointer(cErr))

		msg := "unknown libvips error"
		if cErr != nil {
			msg = C.GoString(cErr)
		}
		return fmt.Errorf("libvips error: %s", msg)
	}

	return nil
}
