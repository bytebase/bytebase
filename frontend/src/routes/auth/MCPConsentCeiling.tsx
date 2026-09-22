import { Check, EyeOff, Info, ScrollText, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mcpRowKey, servedRows } from "@/components/mcp/mcpCapabilityRows";
import type { MCPServingMode } from "@/components/mcp/mcpPolicy";
import { Alert } from "@/components/ui/alert";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPConsentLine } from "./MCPConsentPolicyCard";
import { MCPConsentPolicyCard } from "./MCPConsentPolicyCard";

interface Props {
  /** The ceiling this session runs at, narrowed by the caller. */
  readonly mode: MCPServingMode;
  readonly ignoreMaskingExemptions: boolean;
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
  mode,
  ignoreMaskingExemptions,
  dataMaskingAvailable,
}: Props) {
  const { t } = useTranslation();

  const readWrite = mode === MCPSetting_Capability.READ_WRITE;
  const lines: MCPConsentLine[] = [
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
    // A bound, not a grant: neutral glyph and no mark, so it is not counted
    // among the rows above it. Scoped by mode because Read-only's statement
    // clamp is one rule with one consequence, while Read-write's answer varies
    // by engine and by operation — so that line states the bound and stops.
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
    ...(ignoreMaskingExemptions && dataMaskingAvailable
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
      <MCPConsentPolicyCard
        label={t("oauth2.consent.mcp.title")}
        mode={mode}
        lines={lines}
      />

      {readWrite && (
        <Alert
          variant="warning"
          description={t("oauth2.consent.mcp.write-caution")}
        />
      )}
    </div>
  );
}
