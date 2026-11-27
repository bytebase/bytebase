import {
  BetweenHorizontalEnd,
  ChevronRight,
  Play,
  Replace,
  RotateCcw,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { CopyButton } from "@/components/ui/copy-button";
import { cn } from "@/lib/utils";
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
  executed?: boolean;
}

export function AgentSqlBlock({
  children,
  sql,
  source,
  onRun,
  executed = false,
}: AgentSqlBlockProps) {
  const { t } = useTranslation();
  const status = useAgentStore(() =>
    source ? inspectAgentSqlAction(source).status : undefined
  );
  const replaceStatus = useAgentStore(() =>
    source
      ? inspectAgentSqlAction(source, "replace-selection").status
      : undefined
  );
  const runReady = useAgentStore(() =>
    Boolean(source && onRun && canBeginAgentSqlResponseAction(source))
  );
  const [running, setRunning] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const runLabel = executed
    ? t("agent.sql-block.run-again")
    : t("agent.sql-block.run-query");
  const editHint =
    status === "stale"
      ? t("agent.sql-block.stale")
      : status === "readonly"
        ? t("agent.sql-block.readonly")
        : status === "unavailable"
          ? t("agent.sql-block.unavailable")
          : t("agent.sql-block.autosave");
  const selection = source?.snapshot.editorTarget?.selection;
  const replaceHint =
    replaceStatus === "ready"
      ? editHint
      : replaceStatus === "readonly"
        ? t("agent.sql-block.readonly")
        : replaceStatus === "unavailable"
          ? t("agent.sql-block.unavailable")
          : t("agent.sql-block.stale");
  const runHint = runReady ? undefined : t("agent.sql-block.stale");

  return (
    <Collapsible
      open={executed ? expanded : true}
      onOpenChange={setExpanded}
      className="my-2 min-w-0 max-w-full overflow-hidden rounded-sm border border-block-border bg-background"
      data-agent-sql-block
      data-agent-executed-sql={executed || undefined}
    >
      <div
        className="flex items-center gap-2 border-b border-block-border bg-control-bg px-2 py-1"
        data-agent-sql-toolbar
      >
        {executed ? (
          <CollapsibleTrigger className="w-auto min-w-0 gap-1 text-xs font-medium text-control-light">
            <ChevronRight
              className={cn("size-3 shrink-0", expanded && "rotate-90")}
              aria-hidden="true"
            />
            {t("agent.sql-block.executed-sql")}
          </CollapsibleTrigger>
        ) : (
          <span className="shrink-0 text-xs font-medium text-control-light">
            {t("common.sql")}
          </span>
        )}
        <AgentActionGroup
          role="group"
          aria-label={t("agent.sql-block.actions")}
          className="flex-1"
        >
          <CopyButton
            content={sql}
            label={t("agent.sql-block.copy")}
            appearance="secondary"
            tooltipLayer="agent"
          />
          {source && (
            <>
              <AgentTooltip
                content={`${t("agent.sql-block.insert")}: ${editHint}`}
              >
                <Button
                  type="button"
                  appearance="secondary"
                  size="xs"
                  aria-label={t("agent.sql-block.insert")}
                  disabled={status !== "ready"}
                  aria-description={editHint}
                  onClick={() => applyAgentSqlAction(source, "insert")}
                >
                  <BetweenHorizontalEnd
                    className="size-3.5"
                    aria-hidden="true"
                  />
                </Button>
              </AgentTooltip>
              {selection && selection.start < selection.end && (
                <AgentTooltip
                  content={`${t("agent.sql-block.replace-selection")}: ${replaceHint}`}
                >
                  <Button
                    type="button"
                    appearance="secondary"
                    size="xs"
                    aria-label={t("agent.sql-block.replace-selection")}
                    disabled={replaceStatus !== "ready"}
                    aria-description={replaceHint}
                    onClick={() =>
                      applyAgentSqlAction(source, "replace-selection")
                    }
                  >
                    <Replace className="size-3.5" aria-hidden="true" />
                  </Button>
                </AgentTooltip>
              )}
              {onRun &&
                source.snapshot.connection &&
                source.snapshot.rowLimit !== undefined && (
                  <AgentTooltip
                    content={runHint ? `${runLabel}: ${runHint}` : runLabel}
                  >
                    <Button
                      type="button"
                      appearance="secondary"
                      size="xs"
                      aria-label={runLabel}
                      disabled={!runReady || running}
                      aria-description={runHint}
                      onClick={async () => {
                        if (running) return;
                        const current = getCurrentAgentSqlActionSource(source);
                        if (
                          !current ||
                          !canBeginAgentSqlResponseAction(current)
                        )
                          return;
                        setRunning(true);
                        try {
                          await onRun(current);
                        } finally {
                          setRunning(false);
                        }
                      }}
                    >
                      {executed ? (
                        <RotateCcw className="size-3.5" aria-hidden="true" />
                      ) : (
                        <Play className="size-3.5" aria-hidden="true" />
                      )}
                    </Button>
                  </AgentTooltip>
                )}
            </>
          )}
        </AgentActionGroup>
      </div>
      <CollapsiblePanel>{children}</CollapsiblePanel>
    </Collapsible>
  );
}
