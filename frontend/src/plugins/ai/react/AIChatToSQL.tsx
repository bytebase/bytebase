import { lazy, Suspense } from "react";
import { useAIContext } from "./context";

// `ChatPanel` pulls in the markdown parser (`unified`, `remark-parse`,
// `remark-gfm`) and the React `MonacoEditor` — both heavy enough that
// gating them behind `lazy()` keeps the initial SQL Editor bundle slim
// when the AI side pane is collapsed. Matches the Vue version's
// `await import("./ChatPanel.vue")` deferred-import pattern.
const ChatPanel = lazy(() =>
  import("./ChatPanel").then((m) => ({ default: m.ChatPanel }))
);

/**
 * React port of `plugins/ai/components/AIChatToSQL.vue`.
 *
 * Top-level entry point for the AI chat side pane. Must be rendered
 * inside `<AIContextProvider>`. Returns `null` when the workspace's AI
 * setting is disabled (the host shows nothing).
 */
export function AIChatToSQL() {
  const { aiSetting } = useAIContext();
  if (!aiSetting.enabled) return null;
  return (
    <Suspense fallback={null}>
      <ChatPanel />
    </Suspense>
  );
}
