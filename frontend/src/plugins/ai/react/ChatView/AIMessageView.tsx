import { Loader2, TriangleAlertIcon } from "lucide-react";
import { cn } from "@/react/lib/utils";
import type { Message } from "../../types";
import { Markdown } from "./Markdown/Markdown";

type Props = {
  readonly message: Message;
};

/**
 * React port of `plugins/ai/components/ChatView/AIMessageView.vue`.
 *
 * Three render states keyed off `message.status`:
 *   - DONE → `<Markdown>` of `message.content`, full-width bubble.
 *   - LOADING → small spinner, content-width bubble.
 *   - FAILED → warning icon + error text, capped at 40% width.
 *
 * `codeBlockProps.width: 1.0` because the AI bubble already spans the
 * row, so an embedded code card uses 100% of the bubble width.
 */
export function AIMessageView({ message }: Props) {
  const isDone = message.status === "DONE";
  const isLoading = message.status === "LOADING";
  const isFailed = message.status === "FAILED";

  return (
    <div
      className={cn(
        "border rounded-sm shadow-sm py-1 px-1 bg-gray-50 border-gray-400",
        isDone && "w-full min-w-36",
        isFailed && "max-w-[40%] min-w-36",
        !isDone && !isFailed && "w-auto"
      )}
    >
      {isDone && (
        <Markdown content={message.content} codeBlockProps={{ width: 1.0 }} />
      )}
      {isLoading && (
        <div className="flex items-center">
          <Loader2 className="mx-1 size-[18px] animate-spin" />
        </div>
      )}
      {isFailed && (
        <div className="text-warning flex items-center gap-x-1">
          <TriangleAlertIcon className="inline-block size-4 shrink-0" />
          <span className="text-sm">{message.error}</span>
        </div>
      )}
    </div>
  );
}
