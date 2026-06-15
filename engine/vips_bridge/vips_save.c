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

int vips_has_loader(const char *name) {
  if (strcmp(name, "avif") == 0) {
    return vips_type_find("VipsForeignLoad", "avifload") != 0 ||
           vips_type_find("VipsForeignLoad", "heifload") != 0;
  }
  if (strcmp(name, "webp") == 0) {
    return vips_type_find("VipsForeignLoad", "webpload") != 0;
  }
  return vips_type_find("VipsForeignLoad", name) != 0;
}

int vips_has_saver(const char *name) {
  if (strcmp(name, "avif") == 0) {
    if (vips_type_find("VipsForeignSave", "avifsave") != 0 ||
        vips_type_find("VipsForeignSave", "heifsave") != 0) {
      return 1;
    }
    const char *saver = vips_foreign_find_save("test.avif");
    return saver != NULL;
  }
  if (strcmp(name, "webp") == 0) {
    if (vips_type_find("VipsForeignSave", "webpsave") != 0) {
      return 1;
    }
    const char *saver = vips_foreign_find_save("test.webp");
    return saver != NULL;
  }
  return vips_type_find("VipsForeignSave", name) != 0;
}
