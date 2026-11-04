import { describe, expect, test } from "vitest";
import { runWithConcurrency } from "./concurrency";

describe("runWithConcurrency", () => {
  test("never runs more than the limit at once and keeps result order", async () => {
    let inFlight = 0;
    let peak = 0;
    const release: (() => void)[] = [];
    const pending = runWithConcurrency([1, 2, 3, 4, 5], 2, (item) => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      return new Promise<number>((resolve) => {
        release.push(() => {
          inFlight--;
          resolve(item * 10);
        });
      });
    });
    await Promise.resolve();
    expect(inFlight).toBe(2);
    while (release.length) {
      release.shift()?.();
      await Promise.resolve();
      await Promise.resolve();
    }
    expect(await pending).toEqual([10, 20, 30, 40, 50]);
    expect(peak).toBe(2);
  });

  test("records a rejection as undefined without stopping the rest", async () => {
    const results = await runWithConcurrency(["a", "b", "c"], 3, async (x) => {
      if (x === "b") throw new Error("boom");
      return x.toUpperCase();
    });
    expect(results).toEqual(["A", undefined, "C"]);
  });

  test("handles an empty list and a limit larger than the list", async () => {
    expect(await runWithConcurrency([], 4, async (x) => x)).toEqual([]);
    expect(await runWithConcurrency([1], 4, async (x) => x)).toEqual([1]);
  });
});
