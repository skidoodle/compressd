#include "vips_save.h"
#include <stdlib.h>
#include <string.h>

int vips_save_to_file(VipsImage *image, const char *path) {
  return vips_image_write_to_file(image, path, NULL);
}

char *vips_get_last_error() {
  const char *err = vips_error_buffer();
  if (err == NULL)
    return NULL;

  size_t len = strlen(err) + 1;
  char *res = malloc(len);
  if (res) {
    memcpy(res, err, len);
  }
  vips_error_clear();
  return res;
}
