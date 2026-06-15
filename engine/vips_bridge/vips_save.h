#ifndef VIPS_SAVE_H
#define VIPS_SAVE_H

#include <vips/vips.h>

/**
 * vips_save_to_file saves a VipsImage to a file.
 * It uses vips_image_write_to_file internally, which supports
 * option strings in the path (e.g. "image.avif[Q=75]").
 */
int vips_save_to_file(VipsImage *image, const char *path);

/**
 * vips_get_last_error returns the current vips error buffer
 * as a string and clears it.
 */
char *vips_get_last_error();

#endif
