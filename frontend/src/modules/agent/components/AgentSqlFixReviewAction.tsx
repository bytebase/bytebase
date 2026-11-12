import { ArrowRight, Check, FilePenLine, X } from "lucide-react";
import { lazy, Suspense, useRef, useState, useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { controlSize } from "@/components/ui/styles.stylex";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "../logic/pageHandoff";
import {
  type AgentSqlFixDraft,
  applyAgentSqlFixToTarget,
  getAgentSqlFixIdentity,
  getAgentSqlFixTargetVersion,
  inspectAgentSqlFixTarget,
  isCurrentAgentSqlFixSource,
  notifyAgentSqlFixTargetChanged,
  revealAgentSqlFixTarget,
  subscribeAgentSqlFixTarget,
} from "../logic/sqlFix";
import type { AgentResponseAction } from "../logic/types";
import { type AgentState, useAgentStore } from "../store/agent";
import { AgentPrimaryAction } from "./AgentPrimaryAction";
import {
  AgentDialog,
  AgentDialogClose,
  AgentDialogContent,
  AgentDialogDescription,
  AgentDialogTitle,
} from "./ui/AgentDialog";

const DiffMonaco = lazy(() =>
  import("@/components/monaco/DiffMonaco").then((module) => ({
    default: module.DiffMonaco,
  }))
);

const DIFF_OPTIONS = {
  renderSideBySide: true,
  useInlineViewWhenSpaceIsLimited: true,
  renderSideBySideInlineBreakpoint: 640,
  originalEditable: false,
  scrollBeyondLastLine: false,
  minimap: { enabled: false },
} as const;

function ownsDraft(state: AgentState, chatId: string, draft: AgentSqlFixDraft) {
  const chat = state.getChat(chatId);
  return Boolean(
    state.currentChatId === chatId &&
      draft.chatId === chatId &&
      chat?.capability?.kind === "diagnostic" &&
      isCurrentAgentSqlFixSource(draft.source, chat.capability.resource)
  );
}

function chatAvailable(state: AgentState, chatId: string) {
  const chat = state.getChat(chatId);
  return Boolean(
    chat &&
      !chat.archived &&
      chat.status === "idle" &&
      !state.queuedUserMessagesByChatId[chatId] &&
      !state.queuedUserMessageInFlightByChatId[chatId] &&
      !state.remoteBusyChatIds[chatId]
  );
}

export function AgentSqlFixReviewAction({
  action,
  chatId,
}: {
  action: Extract<AgentResponseAction, { kind: "review-sql-fix" }>;
  chatId: string;
}) {
  const { t } = useTranslation();
  const state = useAgentStore();
  useAppStore(() => getAgentSqlFixIdentity());
  useSyncExternalStore(subscribeAgentSqlFixTarget, getAgentSqlFixTargetVersion);
  const [open, setOpen] = useState(false);
  const [revealing, setRevealing] = useState(false);
  const [rejected, setRejected] = useState(false);
  const applied = useRef(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const candidate = state.sqlFixDraftsById[action.draftId];
  const draft =
    candidate && ownsDraft(state, chatId, candidate) ? candidate : undefined;
  const available = chatAvailable(state, chatId);
  const target = draft ? inspectAgentSqlFixTarget(draft.source) : undefined;
  const currentStatus =
    target?.status === "ready" && target.currentSql !== draft?.source.sql
      ? "stale"
      : target?.status;
  const status = !draft
    ? "original-unavailable"
    : !available
      ? "busy"
      : rejected
        ? "apply-failed"
        : (currentStatus ?? "unavailable");
  const ready = status === "ready" && target?.currentSql === draft?.source.sql;
  const statusMessage = {
    ready: "",
    "original-unavailable": t("agent.sql-fix.original-unavailable"),
    busy: t("agent.sql-fix.busy"),
    "apply-failed": t("agent.sql-fix.apply-failed"),
    stale: t("agent.sql-fix.stale"),
    loading: t("agent.sql-fix.loading"),
    readonly: t("agent.sql-fix.readonly"),
    "permission-denied": t("agent.sql-fix.permission-denied"),
    unavailable: t("agent.sql-fix.unavailable"),
  }[status];

  const review = () => {
    applied.current = false;
    setRejected(false);
    if (!draft) {
      setOpen(true);
      return;
    }
    const identity = getAgentSqlFixIdentity();
    try {
      const reveal = revealAgentSqlFixTarget(draft.source);
      if (reveal instanceof Promise) {
        setRevealing(true);
        void reveal
          .then((revealed) => {
            if (
              revealed !== false &&
              identity === getAgentSqlFixIdentity() &&
              useAgentStore.getState().currentChatId === chatId
            )
              setOpen(true);
          })
          .catch(() => {
            if (
              identity === getAgentSqlFixIdentity() &&
              useAgentStore.getState().currentChatId === chatId
            ) {
              setRejected(true);
              setOpen(true);
            }
          })
          .finally(() => setRevealing(false));
      } else if (reveal !== false) setOpen(true);
    } catch {
      setRejected(true);
      setOpen(true);
    }
  };

  const apply = () => {
    const latest = useAgentStore.getState();
    const currentDraft = latest.sqlFixDraftsById[action.draftId];
    if (
      !currentDraft ||
      !ownsDraft(latest, chatId, currentDraft) ||
      !chatAvailable(latest, chatId)
    ) {
      setRejected(true);
      return;
    }
    const currentTarget = inspectAgentSqlFixTarget(currentDraft.source);
    if (
      currentTarget.status !== "ready" ||
      currentTarget.currentSql !== currentDraft.source.sql
    ) {
      setRejected(false);
      notifyAgentSqlFixTargetChanged();
      return;
    }
    const handoff = captureAgentPageHandoffContext(chatId);
    if (!applyAgentSqlFixToTarget(currentDraft.source, currentDraft.sql)) {
      setRejected(true);
      return;
    }
    applied.current = true;
    setOpen(false);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("agent.sql-fix.applied"),
    });
    minimizeAgentAfterPageHandoff(handoff);
  };

  const reviewCurrent = async () => {
    if (!draft) return;
    const identity = getAgentSqlFixIdentity();
    const handoff = captureAgentPageHandoffContext(chatId);
    setOpen(false);
    try {
      if ((await revealAgentSqlFixTarget(draft.source)) === true) {
        applied.current = true;
        minimizeAgentAfterPageHandoff(handoff);
      }
    } catch {
      if (
        identity === getAgentSqlFixIdentity() &&
        useAgentStore.getState().currentChatId === chatId
      ) {
        setRejected(true);
        setOpen(true);
      }
    }
  };

  if (state.getChat(chatId)?.capability?.kind !== "diagnostic") return null;

  return (
    <>
      <AgentPrimaryAction
        className="-ml-2"
        data-agent-action-review-sql-fix
        disabled={!available}
        loading={revealing}
        onClick={review}
        ref={trigger}
      >
        {t("agent.sql-fix.review")}
      </AgentPrimaryAction>
      <AgentDialog onOpenChange={setOpen} open={open}>
        <AgentDialogContent
          className="w-[calc(100vw-2rem)] max-w-5xl 2xl:max-w-5xl max-h-[calc(100dvh-2rem)]"
          finalFocus={() => (applied.current ? false : trigger.current)}
        >
          <div className="flex items-start justify-between gap-4 border-b border-block-border p-6">
            <div className="flex min-w-0 flex-col gap-1">
              <AgentDialogTitle>{t("agent.sql-fix.review")}</AgentDialogTitle>
              {draft && (
                <AgentDialogDescription className="wrap-anywhere">
                  {`${draft.source.plan} · ${draft.source.specId}`}
                </AgentDialogDescription>
              )}
              {draft && (
                <span className="wrap-anywhere text-xs text-control-light">
                  {draft.source.targets.join(", ")}
                </span>
              )}
            </div>
            <AgentDialogClose
              render={
                <Button
                  appearance="ghost"
                  aria-label={t("common.close")}
                  className="w-7 shrink-0 px-0"
                  size="sm"
                />
              }
            >
              <X size={controlSize.sm.iconSize} />
            </AgentDialogClose>
          </div>
          <div className="flex min-w-0 flex-col gap-4 p-6">
            {draft && (
              <div className="min-w-0">
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-xs text-control">
                  <span>
                    {t("agent.sql-fix.checked-sql")} · SHA256{" "}
                    {draft.source.sha256.slice(0, 12)}
                  </span>
                  <span>{t("agent.sql-fix.proposed-sql")}</span>
                </div>
                <Suspense
                  fallback={
                    <div className="min-h-48 text-sm text-control-light">
                      {t("common.loading")}
                    </div>
                  }
                >
                  <DiffMonaco
                    min={180}
                    max={360}
                    original={draft.source.sql}
                    modified={draft.sql}
                    options={DIFF_OPTIONS}
                    readOnly
                  />
                </Suspense>
              </div>
            )}
            {status !== "ready" && (
              <Alert variant={status === "loading" ? "info" : "warning"}>
                {statusMessage}
              </Alert>
            )}
            {draft && status === "stale" && (
              <Button
                appearance="link"
                className="self-start"
                onClick={reviewCurrent}
                size="sm"
              >
                {t("agent.sql-fix.review-current")}
                <ArrowRight size={controlSize.sm.iconSize} />
              </Button>
            )}
          </div>
          <div className="flex flex-wrap items-center justify-between gap-4 border-t border-block-border p-6">
            <span className="flex items-center gap-1.5 text-xs text-control-light">
              <FilePenLine className="size-4" />
              {t("agent.sql-fix.unsaved-only")}
            </span>
            <div className="ml-auto flex items-center gap-2">
              <AgentDialogClose render={<Button appearance="outline" />}>
                {t("common.cancel")}
              </AgentDialogClose>
              <Button disabled={!ready} onClick={apply}>
                <Check size={controlSize.md.iconSize} />
                {t("agent.sql-fix.apply")}
              </Button>
            </div>
          </div>
        </AgentDialogContent>
      </AgentDialog>
    </>
  );
}
