import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { NumberInput } from "@/components/ui/number-input";
import { Switch } from "@/components/ui/switch";
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
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-3">
      {GHOST_PARAMETERS.map((param) => (
        <GhostFlagRow
          key={param.key}
          param={param}
          value={value}
          onChange={onChange}
          riskCaption={
            param.key === "skip-metadata-lock-check"
              ? {
                  text: t("plan.ghost.skip-metadata-lock-check-risk"),
                  link: t("plan.ghost.skip-metadata-lock-check-why"),
                }
              : undefined
          }
        />
      ))}
    </div>
  );
}

function GhostFlagRow({
  param,
  value,
  onChange,
  riskCaption,
}: {
  param: GhostParameter;
  value: Record<string, string>;
  onChange: (next: Record<string, string>) => void;
  riskCaption?: { text: string; link: string };
}) {
  const current = value[param.key];
  const isEnabled =
    param.type === "bool" && isBooleanFlagEnabled(current, param.default);
  const set = (raw: string | number | boolean | null | undefined) =>
    onChange(withFlag(value, param, raw));

  return (
    <div data-flag={param.key} className="flex flex-col gap-y-1">
      <div className="flex min-h-7 items-center justify-between gap-x-4">
        <span
          className="truncate font-mono text-sm text-control"
          title={param.key}
        >
          {param.key}
        </span>
        {param.type === "bool" ? (
          <Switch
            size="sm"
            checked={isEnabled}
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
      {riskCaption && isEnabled && (
        <Alert
          appearance="caption"
          variant="warning"
          data-testid="skip-metadata-lock-check-risk"
        >
          <span>
            {riskCaption.text}{" "}
            <a
              href="https://github.com/github/gh-ost/pull/1536"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-x-1 text-warning underline"
            >
              {riskCaption.link}
              <ExternalLink className="size-3" />
            </a>
          </span>
        </Alert>
      )}
    </div>
  );
}

const TRUE_BOOLEAN_VALUES = new Set(["1", "t", "T", "true", "TRUE", "True"]);

function isBooleanFlagEnabled(value: string | undefined, defaultValue: string) {
  return TRUE_BOOLEAN_VALUES.has(value ?? defaultValue);
}
