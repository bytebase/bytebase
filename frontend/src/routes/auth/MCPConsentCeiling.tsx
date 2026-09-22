import { Info, ScrollText } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { MCPCapabilityList } from "@/components/mcp/MCPCapabilityList";
import type { MCPServingMode } from "@/components/mcp/mcpPolicy";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  const [details, setDetails] = useState(false);

  const lines: MCPConsentLine[] = [
    // A bound, not a grant: neutral glyph and no mark, so it is not counted
    // among the rows above it. The capability list holds mode-specific details;
    // this shared line gives the constraint that applies to every MCP session.
    {
      key: "capped",
      icon: <Info className="size-4 text-control-light" />,
      text: t("settings.mcp.policy.bound"),
    },
    {
      key: "audit",
      icon: <ScrollText className="size-4 text-control-light" />,
      text: t("settings.mcp.policy.audit"),
    },
  ];

  return (
    <div className="flex flex-col gap-3">
      <MCPConsentPolicyCard
        label={t("oauth2.consent.mcp.title")}
        mode={mode}
        modeAddon={
          ignoreMaskingExemptions && dataMaskingAvailable ? (
            <Badge variant="secondary" className="whitespace-nowrap">
              {t("settings.mcp.policy.masking.badge")}
            </Badge>
          ) : undefined
        }
        headerAction={
          <Button
            appearance="link"
            size="sm"
            aria-expanded={details}
            onClick={() => setDetails(!details)}
          >
            {details
              ? t("settings.mcp.ladder.hide-details")
              : t("settings.mcp.ladder.show-details")}
          </Button>
        }
        lines={lines}
      >
        <MCPCapabilityList mode={mode} details={details} tierDividers floor />
      </MCPConsentPolicyCard>

      {mode === MCPSetting_Capability.READ_WRITE && (
        <Alert
          variant="warning"
          description={t("oauth2.consent.mcp.write-caution")}
        />
      )}
    </div>
  );
}
