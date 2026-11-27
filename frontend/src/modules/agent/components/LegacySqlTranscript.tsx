import type { LegacySqlTranscript as Transcript } from "../logic/legacySqlHistory";
import { AgentChat } from "./AgentChat";

export function LegacySqlTranscript({ chat, messages }: Transcript) {
  return (
    <article
      aria-label={chat.title}
      className="flex min-w-0 flex-col gap-2 p-3"
    >
      <h3 className="break-words font-medium">{chat.title}</h3>
      <p className="break-all text-xs text-control-light">
        {[chat.legacySqlImport?.instance, chat.legacySqlImport?.database]
          .filter(Boolean)
          .join(" / ")}
      </p>
      <AgentChat legacyTranscript={{ chat, messages }} />
    </article>
  );
}
