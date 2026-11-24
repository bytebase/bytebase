import { Info, ScrollText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MCPCapabilityList } from "@/components/mcp/MCPCapabilityList";
import type { MCPServingMode } from "@/components/mcp/mcpPolicy";
import { Alert } from "@/components/ui/alert";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPConsentLine } from "./MCPConsentPolicyCard";
import { MCPConsentPolicyCard } from "./MCPConsentPolicyCard";

interface Props {
  /** The ceiling this session runs at, narrowed by the caller. */
  readonly mode: MCPServingMode;
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
export function MCPConsentCeiling({ mode }: Props) {
  const { t } = useTranslation();

  const lines: MCPConsentLine[] = [
    // A bound, not a grant: neutral glyph and no mark, so it is not counted
    // among the rows above it. Row titles carry no caveats, so this one line
    // bounds them all.
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
        label={t("oauth2.consent.mcp.policy-label")}
        mode={mode}
        lines={lines}
      >
        <MCPCapabilityList
          mode={mode}
          allowedHeading={t("oauth2.consent.mcp.title")}
          refusedHeading={t("oauth2.consent.mcp.title-refused")}
          headingLevel={2}
        />
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
