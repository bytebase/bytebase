import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useAgentStore } from "../store/agent";

export function AgentPageHelpButton({ className }: { className?: string }) {
  const { t } = useTranslation();

  return (
    <Button
      type="button"
      appearance="outline"
      size="md"
      className={className}
      onClick={() => useAgentStore.getState().open()}
    >
      {t("agent.masking.ask-agent", {
        defaultValue: "Ask Agent about masking",
      })}
    </Button>
  );
}
