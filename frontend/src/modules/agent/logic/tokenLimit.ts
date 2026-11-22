// Scale decimal digits directly so large saved token counts remain exact.
export function formatTokenLimit(tokens: number): string {
  const digits = String(tokens).padStart(4, "0");
  const fraction = digits.slice(-3).replace(/0+$/, "");
  return `${digits.slice(0, -3)}${fraction ? `.${fraction}` : ""}`;
}

export function parseTokenLimit(value: string): number | null {
  const match = /^(\d*)(?:\.(\d*))?$/.exec(value.trim());
  if (!match || (!match[1] && !match[2])) return null;
  const fraction = (match[2] ?? "").replace(/0+$/, "");
  if (fraction.length > 3) return null;
  const tokens = Number(`${match[1] || "0"}${fraction.padEnd(3, "0")}`);
  return Number.isSafeInteger(tokens) && tokens > 0 ? tokens : null;
}
