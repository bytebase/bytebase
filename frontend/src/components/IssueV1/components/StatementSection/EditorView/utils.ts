import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";

export const readFileAsync = (
  event: Event,
  maxFileSizeMB: number
): Promise<{
  filename: string;
  content: string;
}> => {
  return new Promise((resolve, reject) => {
    const target = event.target as HTMLInputElement;
    const file = (target.files || [])[0];
    const cleanup = () => {
      // Note that once selected a file, selecting the same file again will not
      // trigger <input type="file">'s change event.
      // So we need to do some cleanup stuff here.
      target.files = null;
      target.value = "";
    };

    if (!file) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "File not found",
      });
      cleanup();
      reject();
      return;
    }
    if (file.size > maxFileSizeMB * 1024 * 1024) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("issue.upload-sql-file-max-size-exceeded", {
          size: `${maxFileSizeMB}MB`,
        }),
      });
      cleanup();
      reject();
      return;
    }

    const fr = new FileReader();
    fr.onload = async () => {
      const content = fr.result as string;
      resolve({
        filename: file.name,
        content: content,
      });
    };
    fr.onerror = () => {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: "Read file error",
        description: String(fr.error),
      });
      reject();
    };
    fr.readAsText(file);
    cleanup();
  });
};
