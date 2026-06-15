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

/**
 * vips_has_loader checks if a loader exists for the given nickname (e.g.
 * "avifload").
 */
int vips_has_loader(const char *name);

/**
 * vips_has_saver checks if a saver exists for the given nickname (e.g.
 * "avifsave").
 */
int vips_has_saver(const char *name);

/**
 * vips_setenv sets an environment variable in the C environment.
 */
void vips_setenv(const char *name, const char *value);

#endif
