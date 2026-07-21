import type { Message } from "../../types";
import { Markdown } from "./Markdown/Markdown";

type Props = {
  readonly message: Message;
};

/**
 * React port of `plugins/ai/components/ChatView/UserMessageView.vue`.
 * User-authored message bubble. Width clamped to 60% of the row so an
 * AI response (`w-full`) and a user prompt visually contrast.
 */
export function UserMessageView({ message }: Props) {
  return (
    <div className="user-message max-w-[60%] min-w-36 border rounded-sm shadow-sm py-1 px-2 bg-indigo-100 border-indigo-400">
      <Markdown content={message.content} codeBlockProps={{ width: 0.6 }} />
    </div>
  );
}
