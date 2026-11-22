import { useTranslation } from "react-i18next";
import {
  type AgentDiagnosticContext,
  buildAgentDiagnosticPrompt,
  openAgentDiagnosticHelp,
} from "../logic/diagnosticHelp";
import { AgentHelpButton } from "./AgentHelpButton";

export function AgentDiagnosticHelpButton({
  context,
}: {
  context: AgentDiagnosticContext;
}) {
  const { t } = useTranslation();
  if (!buildAgentDiagnosticPrompt(context)) return null;
  return (
    <AgentHelpButton size="sm" onClick={() => openAgentDiagnosticHelp(context)}>
      {t("agent.diagnostic-help.ask-agent")}
    </AgentHelpButton>
  );
}
