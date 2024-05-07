import JSZip from "jszip";
import { orderBy } from "lodash-es";
import type { UploadFileInfo } from "naive-ui";
import { defer } from "@/utils";

export type ParsedFile = {
  name: string;
  content: string;
};

const unzip = async (file: File) => {
  const zip = await JSZip.loadAsync(file);
  const files = orderBy(zip.files, (f) => f.name, "asc").filter(
    (f) => !f.dir && f.name.toLowerCase().endsWith(".sql")
  );
  const results = await Promise.all(
    files.map<Promise<ParsedFile>>(async (f) => {
      const content = await f.async("string");
      return {
        name: f.name,
        content,
      };
    })
  );
  return results;
};

const readFile = (file: File) => {
  const d = defer<string>();
  const fr = new FileReader();
  fr.addEventListener("load", (e) => {
    const result = fr.result;
    if (typeof result === "string") {
      d.resolve(result);
      return;
    }
    d.reject(new Error("Failed to read file content."));
  });
  fr.addEventListener("error", (e) => {
    d.reject(fr.error ?? new Error("Failed to read file content."));
  });
  fr.readAsText(file);
  return d.promise;
};

export const readUpload = async (options: { file: UploadFileInfo }) => {
  const fileInfo = options.file;
  if (!fileInfo.file) {
    return [];
  }

  const files: ParsedFile[] = [];
  if (fileInfo.name.toLowerCase().endsWith(".sql")) {
    const content = await readFile(fileInfo.file);
    files.push({
      name: fileInfo.file.name,
      content,
    });
  } else {
    const results = await unzip(fileInfo.file);
    files.push(...results);
  }
  return files;
};
