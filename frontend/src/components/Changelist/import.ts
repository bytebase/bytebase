import JSZip from "jszip";
import { orderBy } from "lodash-es";
import type { UploadFileInfo } from "naive-ui";

export type ParsedFile = {
  name: string;
  arrayBuffer: ArrayBuffer;
};

const unzip = async (file: File) => {
  const zip = await JSZip.loadAsync(file);
  const files = orderBy(zip.files, (f) => f.name, "asc").filter(
    (f) => !f.dir && f.name.toLowerCase().endsWith(".sql")
  );
  const results = await Promise.all(
    files.map<Promise<ParsedFile>>(async (f) => {
      const arrayBuffer = await f.async("arraybuffer");
      return {
        name: f.name,
        arrayBuffer,
      };
    })
  );
  return results;
};

export const readUpload = async (fileInfo: UploadFileInfo) => {
  if (!fileInfo.file) {
    return [];
  }

  const files: ParsedFile[] = [];
  if (fileInfo.name.toLowerCase().endsWith(".sql")) {
    if (!fileInfo.file) {
      // Should not reach here.
      return [];
    }
    const arrayBuffer = await fileInfo.file.arrayBuffer();
    files.push({
      name: fileInfo.file.name,
      arrayBuffer: arrayBuffer,
    });
  } else {
    const results = await unzip(fileInfo.file);
    files.push(...results);
  }
  return files;
};
