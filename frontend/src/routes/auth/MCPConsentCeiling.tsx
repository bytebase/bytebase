import { Check, EyeOff, Info, ScrollText, X } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { MCPModeBadge } from "@/components/mcp/MCPModeBadge";
import { mcpRowKey, servedRows } from "@/components/mcp/mcpCapabilityRows";
import type { MCPServingMode } from "@/components/mcp/mcpPolicy";
import { Alert } from "@/components/ui/alert";
import {
  type MCPSetting,
  MCPSetting_Capability,
} from "@/types/proto-es/v1/setting_service_pb";

interface Line {
  readonly key: string;
  readonly icon: ReactNode;
  /**
   * What the mark means, for a reader who cannot see it. Allowed and refused
   * are otherwise carried by a green check against a red cross — colour and
   * glyph, both visual — on the screen where someone decides whether to hand an
   * agent access.
   */
  readonly mark?: string;
  readonly text: string;
}

interface Props {
  readonly setting: MCPSetting;
  /** The ceiling this session runs at, narrowed by the caller. */
  readonly mode: MCPServingMode;
  readonly dataMaskingAvailable: boolean;
}

/**
 * What the workspace's ceiling lets this session do, shown before the person
 * approves rather than after.
 *
 * The capability rows are the settings page's, from the same table, so the
 * wording an admin chose the policy by is the wording the person approving
 * reads. The same ceiling refuses the POST server-side, so this page is the
 * richer render of a decision the backend makes either way — never the
 * decision itself.
 */
export function MCPConsentCeiling({
  setting,
  mode,
  dataMaskingAvailable,
}: Props) {
  const { t } = useTranslation();

  const readWrite = mode === MCPSetting_Capability.READ_WRITE;
  const lines: Line[] = [
    ...servedRows(mode).map((row) => ({
      key: row.id,
      icon: <Check className="size-4 text-success" />,
      mark: t("settings.mcp.ladder.mark.allowed"),
      text: t(mcpRowKey(row, "title")),
    })),
    // One line for the whole unserved tier rather than five muted rows: this
    // screen is an approval, not a comparison, and the person is deciding
    // about what the session gains.
    ...(readWrite
      ? []
      : [
          {
            // No mark: the sentence states the prohibition itself, and
            // prefixing "Not allowed" would double the negative.
            key: "no-write",
            icon: <X className="size-4 text-error" />,
            text: t("oauth2.consent.mcp.line.no-write"),
          },
        ]),
    // A bound, not a grant: neutral glyph and no mark, so four green checks
    // are not read as four granted capabilities. Scoped by mode because the
    // statement clamp it describes runs only under Read-only — under
    // Read-write the same engines execute unverified instead.
    {
      key: "capped",
      icon: <Info className="size-4 text-control-light" />,
      text: readWrite
        ? t("oauth2.consent.mcp.line.capped-read-write")
        : t("oauth2.consent.mcp.line.capped-read-only"),
    },
    // Both halves, because the line promises a restriction. The toggle
    // withholds unmasking exemptions from MCP sessions, which changes nothing
    // where masking does not run at all — asserting it there would tell the
    // person approving that their data is covered when it is not.
    ...(setting.ignoreMaskingExemptions && dataMaskingAvailable
      ? [
          {
            key: "masking",
            icon: <EyeOff className="size-4 text-control-light" />,
            text: t("oauth2.consent.mcp.line.masking"),
          },
        ]
      : []),
    {
      key: "audit",
      icon: <ScrollText className="size-4 text-control-light" />,
      text: t("oauth2.consent.mcp.line.audit"),
    },
  ];

  return (
    <div className="flex flex-col gap-3">
      <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between gap-x-2">
          <p className="text-sm text-control-light">
            {t("oauth2.consent.mcp.title")}
          </p>
          <MCPModeBadge mode={mode} />
        </div>
        <ul className="text-sm text-main flex flex-col gap-2">
          {lines.map((line) => (
            <li key={line.key} className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0" aria-hidden="true">
                {line.icon}
              </span>
              {line.mark && <span className="sr-only">{line.mark}</span>}
              <span>{line.text}</span>
            </li>
          ))}
        </ul>
      </div>

      {readWrite && (
        <Alert
          variant="warning"
          description={t("oauth2.consent.mcp.write-caution")}
        />
      )}
    </div>
  );
}
