import { useTranslation } from "react-i18next";
import { useAppStore } from "@/stores/app";
import { getAgentAIConfigurationEnabled } from "../logic/aiConfiguration";
import {
  type AgentDiagnosticContext,
  buildAgentDiagnosticPrompt,
  openAgentDiagnosticHelp,
} from "../logic/diagnosticHelp";
import { useAgentStore } from "../store/agent";
import { AgentHelpButton } from "./AgentHelpButton";

export function useAgentDiagnosticHelpAvailable(): boolean {
  const workspaceName = useAppStore((state) => state.workspace?.name);
  const currentUser = useAppStore((state) => state.currentUser);
  const aiEnabled = useAppStore((state) => state.serverInfo?.aiEnabled);
  const aiSetting = useAppStore(
    (state) => state.settingsByName?.["settings/AI"]
  );
  const storedWorkspaceName = useAgentStore((state) => state.workspaceName);
  const storedUserEmail = useAgentStore((state) => state.userEmail);
  const storagePressure = useAgentStore((state) => state.storagePressure);
  const initializationFailureIdentity = useAgentStore(
    (state) => state.initializationFailureIdentity
  );
  const hasInitializedScope = Boolean(storedWorkspaceName || storedUserEmail);
  const ownsScope =
    storedWorkspaceName === workspaceName &&
    storedUserEmail === currentUser?.email;

  return Boolean(
    workspaceName &&
      currentUser?.email &&
      currentUser.workspace === workspaceName &&
      initializationFailureIdentity !==
        JSON.stringify([workspaceName, currentUser.email]) &&
      aiEnabled !== false &&
      getAgentAIConfigurationEnabled(aiSetting) !== false &&
      (!hasInitializedScope || (ownsScope && !storagePressure))
  );
}

export function AgentDiagnosticHelpButton({
  context,
  iconOnly = false,
  className,
}: {
  context: AgentDiagnosticContext;
  iconOnly?: boolean;
  className?: string;
}) {
  const { t } = useTranslation();
  const available = useAgentDiagnosticHelpAvailable();
  if (!available || !buildAgentDiagnosticPrompt(context)) return null;
  return (
    <AgentHelpButton
      size="sm"
      iconOnly={iconOnly}
      className={className}
      onClick={() => openAgentDiagnosticHelp(context)}
    >
      {t("agent.diagnostic-help.ask-agent")}
    </AgentHelpButton>
  );
}
