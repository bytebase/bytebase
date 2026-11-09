import { useTranslation } from "react-i18next";
import { useAgentStore } from "../store/agent";
import { AgentHelpButton } from "./AgentHelpButton";

export function AgentPageHelpButton({ className }: { className?: string }) {
  const { t } = useTranslation();

  return (
    <AgentHelpButton
      className={className}
      onClick={() => useAgentStore.getState().open()}
    >
      {t("agent.masking.ask-agent-short", {
        defaultValue: "Ask Agent",
      })}
    </AgentHelpButton>
  );
}
