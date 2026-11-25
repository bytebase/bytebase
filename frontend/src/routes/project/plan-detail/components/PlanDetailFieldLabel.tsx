// The star marks a required field that is still empty, so it clears once the
// field is filled.
export function PlanDetailFieldLabel({
  label,
  missing,
}: {
  label: string;
  missing: boolean;
}) {
  return (
    <div className="flex items-center gap-x-1 textlabel uppercase">
      <span>{label}</span>
      {missing && <span className="text-error">*</span>}
    </div>
  );
}
