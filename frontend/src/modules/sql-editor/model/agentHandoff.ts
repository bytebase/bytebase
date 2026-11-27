import { isEqual } from "lodash-es";
import { captureAgentPageSnapshot } from "@/modules/agent/logic/capturePage";
import { normalizeAgentChatSnapshot } from "@/modules/agent/logic/sqlEditorContext";
import { useAgentStore } from "@/modules/agent/store/agent";

export type SqlEditorAgentAction =
  | "explain-code"
  | "find-problems"
  | "new-chat";

export function handoffSqlEditorAction(
  action: SqlEditorAgentAction,
  sql: string,
  prompt: string
) {
  const initial = useAgentStore.getState();
  initial.syncState();
  const source = captureAgentPageSnapshot();
  const store = useAgentStore.getState();
  const workspaceName = store.workspaceName;
  if (
    !source?.sqlEditor ||
    !workspaceName ||
    !store.userEmail ||
    store.workspaceName !== initial.workspaceName ||
    store.userEmail !== initial.userEmail
  )
    return false;
  const page = normalizeAgentChatSnapshot({
    ...source,
    sqlEditor: {
      ...source.sqlEditor,
      attachment: {
        kind:
          action === "explain-code"
            ? "explain"
            : action === "new-chat"
              ? "selection"
              : "find-problems",
        sql,
      },
    },
  });
  if (!page?.sqlEditor) return false;
  const context = page.sqlEditor.project ?? workspaceName;
  const fresh = () => store.createChat({ page, context, workspaceName });
  let chat =
    action === "new-chat"
      ? fresh()
      : store.selectOrCreateChatForContext(context, undefined, workspaceName);
  if (!chat) return false;
  const live = useAgentStore.getState();
  if (
    live.draftsByChatId[chat.id] ||
    chat.legacySqlImport ||
    chat.interrupted ||
    chat.runCheckpoint ||
    live.isChatRunning(chat.id)
  )
    chat = fresh();
  if (action === "new-chat") {
    store.setDraft(chat.id, { text: sql, queuedSqlPage: page });
    if (!useAgentStore.getState().getDraft(chat.id).queuedSqlPage) return false;
  } else {
    const enqueue = (chatId: string) => {
      store.queueUserMessage({ chatId, text: prompt, page });
      const current = useAgentStore.getState();
      const message = current.queuedUserMessagesByChatId[chatId];
      return (
        current.workspaceName === workspaceName &&
        current.userEmail === store.userEmail &&
        message?.text === prompt &&
        isEqual(message.page, page)
      );
    };
    if (!enqueue(chat.id)) {
      const current = useAgentStore.getState();
      if (
        current.workspaceName !== workspaceName ||
        current.userEmail !== store.userEmail
      )
        return false;
      chat = fresh();
      if (!enqueue(chat.id)) return false;
    }
  }
  store.updateChatPage(chat.id, page);
  store.open();
  return true;
}
