import { Input } from "@/react/components/ui/input";
import { NumberInput } from "@/react/components/ui/number-input";
import { Switch } from "@/react/components/ui/switch";
import { GHOST_PARAMETERS, type GhostParameter, withFlag } from "./constants";

interface GhostFlagsFormProps {
  /** Current gh-ost flags; only non-default overrides are present. */
  value: Record<string, string>;
  onChange: (next: Record<string, string>) => void;
}

/**
 * The gh-ost parameter controls — one typed control per supported flag. Each
 * control shows the backend default and `onChange` fires with the minimized flag
 * map (only non-default overrides); the parent decides when to persist it.
 */
export function GhostFlagsForm({ value, onChange }: GhostFlagsFormProps) {
  return (
    <div className="flex flex-col gap-y-3">
      {GHOST_PARAMETERS.map((param) => (
        <GhostFlagRow
          key={param.key}
          param={param}
          value={value}
          onChange={onChange}
        />
      ))}
    </div>
  );
}

function GhostFlagRow({
  param,
  value,
  onChange,
}: {
  param: GhostParameter;
  value: Record<string, string>;
  onChange: (next: Record<string, string>) => void;
}) {
  const current = value[param.key];
  const set = (raw: string | number | boolean | null | undefined) =>
    onChange(withFlag(value, param, raw));

  return (
    <div
      data-flag={param.key}
      className="flex min-h-7 items-center justify-between gap-x-4"
    >
      <span
        className="truncate font-mono text-sm text-control"
        title={param.key}
      >
        {param.key}
      </span>
      {param.type === "bool" ? (
        <Switch
          size="sm"
          checked={
            current !== undefined
              ? current === "true"
              : param.default === "true"
          }
          onCheckedChange={(checked) => set(checked)}
        />
      ) : param.type === "string" ? (
        <Input
          size="sm"
          className="w-48"
          aria-label={param.key}
          placeholder={param.default || param.key}
          value={current ?? ""}
          onChange={(e) => set(e.target.value)}
        />
      ) : (
        <NumberInput
          size="sm"
          className="w-48"
          aria-label={param.key}
          placeholder={param.default}
          value={current !== undefined ? Number(current) : null}
          step={param.type === "float" ? 0.1 : 1}
          onValueChange={(v) => set(v)}
        />
      )}
    </div>
  );
}
