import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { Alert } from "@/components/ui/alert";
import { useAppStore } from "@/stores/app";
import {
  assertCurrentContext,
  assertCurrentTarget,
  waitForTarget,
} from "../logic/draftHandoff";
import { resolveAgentHandoffRoute } from "../logic/handoff";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "../logic/pageHandoff";
import {
  consumeSemanticTypeDraft,
  getSafeSemanticTypeDraftSummary,
  getSemanticTypeDraft,
  getSemanticTypeDraftSource,
  revalidateSemanticTypeDraft,
} from "../logic/semanticTypeDraft";
import {
  isSemanticTypeDraftTargetReady,
  openSemanticTypeDraftInEditor,
} from "../logic/semanticTypeDraftTarget";
import { useAgentStore } from "../store/agent";
import { AgentPrimaryAction } from "./AgentPrimaryAction";

export function SemanticTypeDraftButton({
  draftId,
  chatId,
  messageId,
}: {
  draftId: string;
  chatId: string;
  messageId: string;
}) {
  const { t } = useTranslation();
  const userName = useAppStore((state) => state.currentUser?.name);
  const workspaceName = useAppStore((state) => state.workspaceResourceName());
  const ownsSelectedChat = useAgentStore(
    (state) =>
      state.currentChatId === chatId &&
      state.getChat(chatId)?.workspaceName === workspaceName
  );
  const completed = useAgentStore((state) => {
    if (!ownsSelectedChat || !userName || !workspaceName) return undefined;
    const handoff = state
      .getMessages(chatId)
      .find((message) => message.id === messageId)
      ?.metadata?.semanticTypeDraftHandoff;
    return handoff?.draftId === draftId
      ? handoff.created
        ? "created"
        : handoff.outcome
      : undefined;
  });
  const [loading, setLoading] = useState(false);
  const [unavailable, setUnavailable] = useState(false);
  const inFlight = useRef(false);
  const context =
    userName && workspaceName && ownsSelectedChat
      ? { workspaceName, userName, chatId }
      : undefined;
  const draft = context && getSemanticTypeDraft(draftId, context);
  const summary = draft && getSafeSemanticTypeDraftSummary(draft);
  const algorithmLabel = summary
    ? summary.algorithm.kind === "inner-outer-mask"
      ? t(
          `settings.sensitive-data.algorithms.inner-outer-mask.${summary.algorithm.type}-mask`
        )
      : t(`settings.sensitive-data.algorithms.${summary.algorithm.kind}.self`)
    : undefined;
  const label =
    completed === "created"
      ? t("agent.semantic-type-draft.created")
      : completed === "opened"
        ? t("agent.semantic-type-draft.opened")
        : t("agent.semantic-type-draft.open");

  const openDraft = async () => {
    if (inFlight.current || completed) return;
    const handoffContext = captureAgentPageHandoffContext(chatId);
    inFlight.current = true;
    setLoading(true);
    setUnavailable(false);
    try {
      if (!context || !getSemanticTypeDraft(draftId, context))
        throw new Error("The semantic type draft is unavailable.");
      assertCurrentContext(context);
      const expected = getSemanticTypeDraftSource(draftId, context);
      const validated = await revalidateSemanticTypeDraft(draftId, context);
      assertCurrentContext(context);
      const target = resolveAgentHandoffRoute({ kind: "open-semantic-types" });
      if (!target) throw new Error("The semantic type editor is unavailable.");
      const targetPath = router.resolve(target).fullPath;
      if (expected) {
        assertCurrentTarget(targetPath);
      } else if (router.currentRoute.value.fullPath !== targetPath) {
        await router.push(target);
      }
      await waitForTarget(context, targetPath, isSemanticTypeDraftTargetReady);
      assertCurrentContext(context);
      assertCurrentTarget(targetPath);
      await openSemanticTypeDraftInEditor(
        validated,
        context,
        { draftId, messageId },
        expected
      );
      assertCurrentContext(context);
      assertCurrentTarget(targetPath);
      useAgentStore
        .getState()
        .recordSemanticTypeDraftHandoff(chatId, messageId, draftId, "opened");
      consumeSemanticTypeDraft(draftId, context);
      minimizeAgentAfterPageHandoff(handoffContext);
    } catch {
      setUnavailable(true);
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  };

  return (
    <div className="flex min-w-0 flex-col items-start gap-2">
      {summary && (
        <dl className="flex max-w-full min-w-0 flex-col gap-1 text-xs">
          <div className="flex flex-wrap items-baseline gap-x-2">
            <dt className="text-control-light">
              {t("settings.sensitive-data.semantic-types.table.title")}
            </dt>
            <dd className="min-w-0 wrap-anywhere">{summary.title}</dd>
          </div>
          {summary.description && (
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">
                {t("settings.sensitive-data.semantic-types.table.description")}
              </dt>
              <dd className="min-w-0 wrap-anywhere">{summary.description}</dd>
            </div>
          )}
          <div className="flex flex-wrap items-baseline gap-x-2">
            <dt className="text-control-light">
              {t(
                "settings.sensitive-data.semantic-types.table.masking-algorithm"
              )}
            </dt>
            <dd className="min-w-0 wrap-anywhere">{algorithmLabel}</dd>
          </div>
        </dl>
      )}
      <AgentPrimaryAction
        className="-ml-2"
        data-agent-action-draft-semantic-type
        loading={loading}
        completed={Boolean(completed)}
        disabled={!draft}
        onClick={() => void openDraft()}
      >
        {label}
      </AgentPrimaryAction>
      <p className="text-xs text-control-light">
        {completed === "created"
          ? t("agent.semantic-type-draft.saved")
          : !draft && !completed && !unavailable
            ? t("agent.semantic-type-draft.inactive")
            : t("agent.semantic-type-draft.unsaved")}
      </p>
      {!completed && unavailable && (
        <Alert appearance="caption" variant="error">
          {t("agent.semantic-type-draft.unavailable")}
        </Alert>
      )}
    </div>
  );
}
