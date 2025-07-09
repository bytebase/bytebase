// Default maximum size of a statement in bytes (100MB).
export const MAX_UPLOAD_FILE_SIZE_MB = 100;

export const getStatementSize = (statement: string): bigint => {
  return BigInt(new TextEncoder().encode(statement).length);
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
