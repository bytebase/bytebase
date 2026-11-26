import { BetweenHorizontalEnd, Play, Replace } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import {
  type AgentSqlActionSource,
  applyAgentSqlAction,
  canBeginAgentSqlResponseAction,
  getCurrentAgentSqlActionSource,
  inspectAgentSqlAction,
} from "../logic/sqlResponse";
import { useAgentStore } from "../store/agent";
import { AgentActionGroup } from "./AgentActionGroup";
import { AgentTooltip } from "./ui/AgentTooltip";

interface AgentSqlBlockProps {
  children: ReactNode;
  sql: string;
  source?: AgentSqlActionSource;
  onRun?: (source: AgentSqlActionSource) => Promise<boolean>;
}

export function AgentSqlBlock({
  children,
  sql,
  source,
  onRun,
}: AgentSqlBlockProps) {
  const { t } = useTranslation();
  const status = useAgentStore(() =>
    source ? inspectAgentSqlAction(source).status : undefined
  );
  const runReady = useAgentStore(() =>
    Boolean(source && onRun && canBeginAgentSqlResponseAction(source))
  );
  const [running, setRunning] = useState(false);
  const editHint =
    status === "stale"
      ? t("agent.sql-block.stale")
      : status === "readonly"
        ? t("agent.sql-block.readonly")
        : status === "unavailable"
          ? t("agent.sql-block.unavailable")
          : t("agent.sql-block.autosave");
  const selection = source?.snapshot.editorTarget?.selection;
  const runHint = runReady ? undefined : t("agent.sql-block.stale");

  return (
    <div className="my-1 min-w-0 max-w-full" data-agent-sql-block>
      {children}
      <AgentActionGroup role="group" aria-label={t("agent.sql-block.actions")}>
        <CopyButton
          content={sql}
          label={t("agent.sql-block.copy")}
          appearance="ghost"
          tooltipLayer="agent"
        />
        {source && (
          <>
            <AgentTooltip content={editHint}>
              <Button
                type="button"
                appearance="secondary"
                size="xs"
                disabled={status !== "ready"}
                aria-description={editHint}
                onClick={() => applyAgentSqlAction(source, "insert")}
              >
                <BetweenHorizontalEnd />
                {t("agent.sql-block.insert")}
              </Button>
            </AgentTooltip>
            {selection && selection.start < selection.end && (
              <AgentTooltip content={editHint}>
                <Button
                  type="button"
                  appearance="secondary"
                  size="xs"
                  disabled={status !== "ready"}
                  aria-description={editHint}
                  onClick={() =>
                    applyAgentSqlAction(source, "replace-selection")
                  }
                >
                  <Replace />
                  {t("agent.sql-block.replace-selection")}
                </Button>
              </AgentTooltip>
            )}
            {onRun &&
              source.snapshot.connection &&
              source.snapshot.rowLimit !== undefined && (
                <AgentTooltip
                  content={runHint ?? t("agent.sql-block.run-query")}
                >
                  <Button
                    type="button"
                    appearance="secondary"
                    size="xs"
                    disabled={!runReady || running}
                    aria-description={runHint}
                    onClick={async () => {
                      if (running) return;
                      const current = getCurrentAgentSqlActionSource(source);
                      if (!current || !canBeginAgentSqlResponseAction(current))
                        return;
                      setRunning(true);
                      try {
                        await onRun(current);
                      } finally {
                        setRunning(false);
                      }
                    }}
                  >
                    <Play />
                    {t("agent.sql-block.run-query")}
                  </Button>
                </AgentTooltip>
              )}
          </>
        )}
      </AgentActionGroup>
    </div>
  );
}
