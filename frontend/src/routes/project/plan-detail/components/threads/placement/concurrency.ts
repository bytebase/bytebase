// Runs `task` over `items` with at most `limit` in flight, preserving order in
// the result. Rejections are collected as `undefined` so one failed sheet
// fetch does not abort the others.
export async function runWithConcurrency<T, R>(
  items: readonly T[],
  limit: number,
  task: (item: T) => Promise<R>
): Promise<(R | undefined)[]> {
  const results: (R | undefined)[] = new Array(items.length);
  let next = 0;
  const worker = async () => {
    while (next < items.length) {
      const index = next++;
      try {
        results[index] = await task(items[index]);
      } catch {
        results[index] = undefined;
      }
    }
  };
  await Promise.all(
    Array.from({ length: Math.max(1, Math.min(limit, items.length)) }, worker)
  );
  return results;
}
