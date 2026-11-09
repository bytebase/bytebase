import { useTranslation } from "react-i18next";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useAgentStore } from "../store/agent";
import { AgentPrimaryActionLink } from "./AgentPrimaryAction";

export function AgentAIConfigurationRecovery({
  onDismiss,
}: {
  onDismiss?: () => void;
}) {
  const { t } = useTranslation();
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const clearError = useAgentStore((s) => s.clearError);
  const allowConfigure = hasWorkspacePermissionV2("bb.settings.set");

  const goConfigure = () => {
    clearError(currentChatId);
    useAgentStore.getState().toggle();
  };

  const dismiss = () => {
    onDismiss?.();
    clearError(currentChatId);
  };

  return (
    <Alert
      variant="warning"
      className="max-w-[80%]"
      title={t("agent.ai-not-configured.title")}
      description={t("agent.ai-not-configured.description")}
    >
      <div className="mt-3 flex flex-wrap gap-x-2 gap-y-2">
        {allowConfigure && (
          <AgentPrimaryActionLink
            to={{
              name: SETTING_ROUTE_WORKSPACE_GENERAL,
              hash: "#ai-assistant",
              query: {
                [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO,
              },
            }}
            onClick={goConfigure}
          >
            {t("agent.ai-not-configured.configure")}
          </AgentPrimaryActionLink>
        )}
        {onDismiss && (
          <Button appearance="outline" size="sm" onClick={dismiss}>
            {t("common.dismiss")}
          </Button>
        )}
      </div>
      {!allowConfigure && (
        <div className="mt-1 text-warning">
          {t("agent.ai-not-configured.contact-admin")}
        </div>
      )}
    </Alert>
  );
}
