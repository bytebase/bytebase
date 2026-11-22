import { useTranslation } from "react-i18next";
import type { AgentApiExecutionEvidence } from "../logic/apiExecution";

export function AgentApiOutcome({
  execution,
}: {
  execution?: AgentApiExecutionEvidence;
}) {
  const { t } = useTranslation();
  if (execution?.result === "not-sent") return t("agent.api-recovery.not-sent");
  switch (execution?.effects) {
    case "acknowledged":
      return t("agent.api-recovery.acknowledged");
    case "partial":
      return t("agent.api-recovery.partial");
    case "none":
      return execution.reason === "import-none-created"
        ? t("agent.api-recovery.none-created")
        : t("agent.api-recovery.rejected");
    case "possible":
      if (execution.result === "failed")
        return t("agent.api-recovery.failed-possible");
      if (execution.result === "succeeded")
        return t("agent.api-recovery.possible");
  }
  return t("agent.api-recovery.unknown");
}
