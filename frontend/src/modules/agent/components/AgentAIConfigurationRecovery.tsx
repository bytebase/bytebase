import { useTranslation } from "react-i18next";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import { Alert } from "@/components/ui/alert";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useAgentStore } from "../store/agent";
import { AgentActionGroup } from "./AgentActionGroup";
import { AgentPrimaryActionLink } from "./AgentPrimaryAction";

export function AgentAIConfigurationRecovery() {
  const { t } = useTranslation();
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const clearError = useAgentStore((s) => s.clearError);
  const allowConfigure = hasWorkspacePermissionV2("bb.settings.set");

  const goConfigure = () => {
    clearError(currentChatId);
    useAgentStore.getState().toggle();
  };

  return (
    <Alert
      variant="warning"
      className="gap-x-2 p-2 sm:gap-x-3 sm:px-4 sm:py-3"
      title={t("agent.ai-not-configured.title")}
      description={t("agent.ai-not-configured.description")}
      data-agent-ai-configuration
    >
      {!allowConfigure && (
        <div className="mt-1 text-warning">
          {t("agent.ai-not-configured.contact-admin")}
        </div>
      )}
      {allowConfigure && (
        <AgentActionGroup
          className="mt-2 sm:mt-3"
          data-agent-ai-configuration-actions
        >
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
        </AgentActionGroup>
      )}
    </Alert>
  );
}
