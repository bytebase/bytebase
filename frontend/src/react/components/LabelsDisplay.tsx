/**
 * Displays label key:value pairs with an overflow count.
 */
export function LabelsDisplay({
  labels,
  showCount = 1,
}: {
  labels: { [key: string]: string };
  showCount?: number;
}) {
  const entries = Object.entries(labels);
  if (entries.length === 0)
    return <span className="text-control-placeholder">-</span>;
  const displayEntries = entries.slice(0, showCount);
  const remaining = entries.length - showCount;
  return (
    <div className="flex items-center gap-x-1">
      {displayEntries.map(([key, value]) => (
        <span key={key} className="rounded-xs bg-gray-100 py-0.5 px-2 text-sm">
          {key}:{value}
        </span>
      ))}
      {remaining > 0 && (
        <span className="text-control-placeholder">+{remaining}</span>
      )}
    </div>
  );
}
