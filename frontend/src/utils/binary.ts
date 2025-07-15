/**
 * Concatenates multiple Uint8Array chunks into a single Uint8Array
 * @param chunks Array of Uint8Array chunks to concatenate
 * @returns A single Uint8Array containing all the data
 */
export const concatUint8Arrays = (chunks: Uint8Array[]): Uint8Array => {
  if (chunks.length === 0) {
    return new Uint8Array();
  }

  if (chunks.length === 1) {
    return chunks[0];
  }

  // Calculate total size and create combined array
  const totalSize = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
  const combined = new Uint8Array(totalSize);

  // Copy chunks into combined array
  chunks.reduce((offset, chunk) => {
    combined.set(chunk, offset);
    return offset + chunk.length;
  }, 0);

  return combined;
};
