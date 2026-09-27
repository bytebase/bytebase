import { ScrollText, X } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import { MCPConsentPolicyCard } from "./MCPConsentPolicyCard";

interface Props {
  readonly workspaceTitle: string;
  /** The workspace row, so a SaaS user can switch to one that permits MCP. */
  readonly workspaceCard: ReactNode;
  readonly onDismiss: () => void;
  readonly dismissing: boolean;
}

/**
 * The consent a disabled workspace never grants.
 *
 * This stops the person before they press Approve. Dismissing posts a denial
 * rather than walking history back, so the client waiting on the callback gets
 * access_denied instead of nothing; that path returns before the ceiling check
 * (backend/api/oauth2/authorize.go), so it writes no ceiling-refusal row. The
 * audit line here is about an attempt that reaches the grant path anyway,
 * which the backend refuses and records (backend/api/oauth2/consent_audit.go).
 */
export function MCPConsentDisabled({
  workspaceTitle,
  workspaceCard,
  onDismiss,
  dismissing,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-6">
      <div className="text-center flex flex-col gap-2">
        <h1 className="text-xl font-semibold text-main">
          {t("oauth2.consent.mcp.disabled.title")}
        </h1>
        <p className="text-control">
          {/* The workspace load is fire-and-forget and swallows its error, so
              the title can be empty. Naming a workspace with nothing in the
              blank reads worse than not naming one. */}
          {workspaceTitle
            ? t("oauth2.consent.mcp.disabled.description", {
                workspace: workspaceTitle,
              })
            : t("oauth2.consent.mcp.disabled.description-no-workspace")}
        </p>
      </div>

      {workspaceCard}

      <MCPConsentPolicyCard
        label={t("oauth2.consent.mcp.disabled.policy-label")}
        mode={MCPSetting_Capability.DISABLED}
        lines={[
          {
            key: "no-session",
            icon: <X className="size-4 text-error" />,
            text: t("oauth2.consent.mcp.disabled.line.no-session"),
          },
          {
            key: "recorded",
            icon: <ScrollText className="size-4 text-control-light" />,
            text: t("oauth2.consent.mcp.disabled.line.recorded"),
          },
        ]}
      />

      <div className="flex flex-col gap-2">
        <Button
          appearance="outline"
          size="lg"
          disabled={dismissing}
          onClick={onDismiss}
        >
          {t("common.close")}
        </Button>
        <p className="text-xs text-control-light text-center">
          {t("oauth2.consent.mcp.disabled.ask-admin")}
        </p>
      </div>
    </div>
  );
}
