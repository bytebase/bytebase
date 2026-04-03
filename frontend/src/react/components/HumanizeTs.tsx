function formatRelativeTime(tsMs: number): string {
  const seconds = Math.floor((Date.now() - tsMs) / 1000);
  const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  if (seconds < 60) return rtf.format(-seconds, "second");
  if (seconds < 3600) return rtf.format(-Math.floor(seconds / 60), "minute");
  if (seconds < 86400) return rtf.format(-Math.floor(seconds / 3600), "hour");
  return rtf.format(-Math.floor(seconds / 86400), "day");
}

interface HumanizeTsProps {
  ts: number;
  className?: string;
}

export function HumanizeTs({ ts, className }: HumanizeTsProps) {
  return <span className={className}>{formatRelativeTime(ts * 1000)}</span>;
}
