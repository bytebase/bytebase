import Long from "long";

// MAX_UPLOAD_FILE_SIZE_MB is the maximum size of the file that can be uploaded in MB.
export const MAX_UPLOAD_FILE_SIZE_MB = 1;

export const getStatementSize = (statement: string): Long => {
  return Long.fromNumber(new TextEncoder().encode(statement).length);
};

export const readFileAsArrayBuffer = (
  file: File
): Promise<{
  filename: string;
  arrayBuffer: ArrayBuffer;
}> => {
  return new Promise((resolve, reject) => {
    const fr = new FileReader();
    fr.onload = async () => {
      const arrayBuffer = fr.result as ArrayBuffer;
      resolve({
        filename: file.name,
        arrayBuffer: arrayBuffer,
      });
    };
    fr.onerror = () => {
      reject(String(fr.error));
    };
    fr.readAsArrayBuffer(file);
  });
};
