import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import {
  type AgentApiAttempt,
  getAgentApiAttemptStatus,
} from "../logic/apiAttempt";
import { useAgentStore } from "../store/agent";
import { AgentApiOutcome } from "./AgentApiOutcome";

export function AgentApiRecovery({ attempt }: { attempt: AgentApiAttempt }) {
  const { t } = useTranslation();
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const [dispatch, setDispatch] = useState<
    "in-flight" | "unknown" | "unavailable"
  >();

  useEffect(() => {
    if (attempt.phase !== "dispatching") return;
    let current = true;
    let timer: ReturnType<typeof setTimeout>;
    const probe = async () => {
      const status = await getAgentApiAttemptStatus(attempt);
      if (!current) return;
      setDispatch(
        status === "in-flight"
          ? status
          : status === "unknown"
            ? status
            : "unavailable"
      );
      // A tab can release its action lease without writing another journal revision.
      if (status === "in-flight") timer = setTimeout(probe, 1000);
    };
    void probe();
    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [attempt]);

  const unresolvedDispatch =
    attempt.phase === "dispatching" && dispatch !== "unknown";
  if (currentChatId !== attempt.chatId) return null;
  return (
    <Alert
      variant="warning"
      showIcon={false}
      data-agent-api-recovery
      role="region"
      aria-label={t("agent.api-recovery.title")}
    >
      <div className="flex min-w-0 flex-col items-start gap-2 text-sm text-main">
        <div>
          <p className="font-medium">
            {unresolvedDispatch ? (
              dispatch === "in-flight" ? (
                t("agent.api-recovery.in-flight")
              ) : (
                t("agent.api-recovery.verifying")
              )
            ) : (
              <AgentApiOutcome execution={attempt.execution} />
            )}
          </p>
          {attempt.operationId !== "*" && (
            <p className="font-mono text-xs text-control-light wrap-anywhere">
              {attempt.operationId}
            </p>
          )}
        </div>
        <p>{t("agent.api-recovery.no-replay")}</p>
      </div>
    </Alert>
  );
}
