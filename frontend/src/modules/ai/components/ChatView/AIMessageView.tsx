import { Loader2, TriangleAlertIcon } from "lucide-react";
import { cn } from "@/lib/utils";
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
        "min-w-0 max-w-full border rounded-sm shadow-sm py-1 px-1 bg-control-bg/80 border-control-border text-control",
        isDone && "w-full",
        isFailed && "max-w-[80%] bg-warning/10 border-warning/60",
        !isDone && !isFailed && "w-auto"
      )}
    >
      {isDone && (
        <Markdown content={message.content} codeBlockProps={{ width: 1.0 }} />
      )}
      {isLoading && (
        <div className="flex items-center text-control-light">
          <Loader2 className="mx-1 size-[18px] animate-spin" />
        </div>
      )}
      {isFailed && (
        <div className="flex min-w-0 items-start gap-x-1.5">
          <TriangleAlertIcon className="mt-0.5 inline-block size-4 shrink-0 text-warning" />
          <span className="min-w-0 wrap-anywhere whitespace-pre-wrap text-sm text-control">
            {message.error}
          </span>
        </div>
      )}
    </div>
  );
}
