import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { Alert } from "@/components/ui/alert";
import { useAppStore } from "@/stores/app";
import { resolveAgentHandoffRoute } from "../logic/handoff";
import {
  consumeMaskingRuleDraft,
  getMaskingRuleDraftSource,
  getMaskingRuleDrafts,
  type MaskingRuleDraftContext,
  revalidateMaskingRuleDraft,
  revalidateMaskingRuleDrafts,
} from "../logic/maskingRuleDraft";
import {
  applyMaskingRuleDraftInEditor,
  isMaskingRuleDraftTargetReady,
  openMaskingRuleDraftInEditor,
  openMaskingRuleDraftsInEditor,
} from "../logic/maskingRuleDraftTarget";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "../logic/pageHandoff";
import { useAgentStore } from "../store/agent";
import { AgentPrimaryAction } from "./AgentPrimaryAction";

function assertCurrentContext(context: MaskingRuleDraftContext): void {
  const app = useAppStore.getState();
  const agent = useAgentStore.getState();
  if (
    app.currentUser?.name !== context.userName ||
    app.workspaceResourceName() !== context.workspaceName ||
    agent.currentChatId !== context.chatId ||
    agent.getChat(context.chatId)?.workspaceName !== context.workspaceName
  ) {
    throw new Error("The masking rule draft context is no longer current.");
  }
}

function assertCurrentTarget(targetPath: string): void {
  if (router.currentRoute.value.fullPath !== targetPath)
    throw new Error("The masking rule editor route is no longer current.");
}

async function waitForTarget(
  context: MaskingRuleDraftContext,
  targetPath: string
): Promise<void> {
  for (let attempt = 0; attempt < 60; attempt++) {
    assertCurrentContext(context);
    assertCurrentTarget(targetPath);
    if (isMaskingRuleDraftTargetReady(context)) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error("The masking rule editor is not available.");
}

export function MaskingRuleDraftButton({
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
  const [loading, setLoading] = useState(false);
  const [unavailable, setUnavailable] = useState(false);
  const completed = useAgentStore((state) => {
    if (!ownsSelectedChat || !userName || !workspaceName) return undefined;
    const handoff = state
      .getMessages(chatId)
      .find((message) => message.id === messageId)
      ?.metadata?.maskingRuleDraftHandoff;
    return handoff?.draftId === draftId ? handoff.outcome : undefined;
  });
  const inFlight = useRef(false);
  const context =
    userName && workspaceName && ownsSelectedChat
      ? { workspaceName, userName, chatId }
      : undefined;
  const drafts = context && getMaskingRuleDrafts(draftId, context);
  const isBatch = Boolean(drafts && drafts.length > 1);
  const source = context && getMaskingRuleDraftSource(draftId, context);
  const label =
    completed === "opened"
      ? t("agent.masking-rule-draft.opened")
      : completed === "applied"
        ? t("agent.masking-rule-draft.applied")
        : source
          ? t("agent.masking-rule-draft.apply")
          : isBatch
            ? t("agent.masking-rule-draft.review-many", {
                count: drafts?.length,
              })
            : t("agent.masking-rule-draft.open");

  const openDraft = async () => {
    if (inFlight.current || completed) return;
    const handoffContext = captureAgentPageHandoffContext(chatId);
    inFlight.current = true;
    setLoading(true);
    setUnavailable(false);
    try {
      if (!context || !getMaskingRuleDrafts(draftId, context)?.length)
        throw new Error("The masking rule draft is unavailable.");
      assertCurrentContext(context);
      const expected = getMaskingRuleDraftSource(draftId, context);
      if (isBatch && expected)
        throw new Error("Rule batches cannot replace an existing draft.");
      const validatedDrafts = isBatch
        ? await revalidateMaskingRuleDrafts(draftId, context)
        : [await revalidateMaskingRuleDraft(draftId, context)];
      const validatedDraft = validatedDrafts[0];
      if (!validatedDraft)
        throw new Error("The masking rule draft is unavailable.");
      assertCurrentContext(context);
      if (expected) {
        await applyMaskingRuleDraftInEditor(validatedDraft, expected, context);
      } else {
        const target = resolveAgentHandoffRoute({
          kind: "open-global-masking",
        });
        if (!target) throw new Error("The masking rule editor is unavailable.");
        const targetPath = router.resolve(target).fullPath;
        await router.push(target);
        await waitForTarget(context, targetPath);
        assertCurrentContext(context);
        assertCurrentTarget(targetPath);
        if (isBatch) {
          await openMaskingRuleDraftsInEditor(validatedDrafts, context);
        } else {
          await openMaskingRuleDraftInEditor(validatedDraft, context);
        }
      }
      assertCurrentContext(context);
      useAgentStore
        .getState()
        .recordMaskingRuleDraftHandoff(
          chatId,
          messageId,
          draftId,
          expected ? "applied" : "opened"
        );
      consumeMaskingRuleDraft(draftId, context);
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
      {drafts?.map((draft, index) => (
        <div key={index} className="flex max-w-full min-w-0 flex-col gap-2">
          {isBatch && (
            <p className="text-xs font-medium wrap-anywhere">{draft.title}</p>
          )}
          <dl className="flex max-w-full min-w-0 flex-col gap-1 text-xs">
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">
                {t("agent.masking-rule-draft.scope")}
              </dt>
              <dd className="min-w-0 wrap-anywhere">
                {draft.scope.kind === "workspace"
                  ? t("agent.masking-rule-draft.all-databases")
                  : draft.scope.kind === "project"
                    ? draft.scope.project
                    : draft.scope.database}
              </dd>
            </div>
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">
                {t("agent.masking-rule-draft.semantic-type")}
              </dt>
              <dd className="min-w-0 wrap-anywhere">{draft.semanticType}</dd>
            </div>
          </dl>
          <code className="max-h-24 max-w-full overflow-y-auto whitespace-pre-wrap text-xs wrap-anywhere">
            {draft.condition}
          </code>
        </div>
      ))}
      <AgentPrimaryAction
        className="-ml-2"
        data-agent-action-draft-masking-rule
        actionKind={source ? "apply" : "open"}
        loading={loading}
        completed={Boolean(completed)}
        disabled={!drafts?.length}
        onClick={() => void openDraft()}
      >
        {label}
      </AgentPrimaryAction>
      <p className="text-xs text-control-light">
        {!drafts?.length && !completed && !unavailable
          ? t("agent.masking-rule-draft.inactive")
          : t("agent.masking-rule-draft.unsaved")}
      </p>
      {!completed && unavailable && (
        <Alert appearance="caption" variant="error">
          {t("agent.masking-rule-draft.unavailable")}
        </Alert>
      )}
    </div>
  );
}
