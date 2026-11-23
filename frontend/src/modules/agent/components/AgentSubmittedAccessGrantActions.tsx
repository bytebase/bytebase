import { decodeSubmittedAccessGrant } from "../logic/submittedAccessGrant";
import type { AgentSubmittedAccessGrant } from "../logic/types";
import { AgentActionReceiptActions } from "./AgentActionReceiptActions";

export function AgentSubmittedAccessGrantActions({
  chatId,
  messageId,
  receipt,
}: {
  chatId: string;
  messageId: string;
  receipt: AgentSubmittedAccessGrant;
}) {
  if (!decodeSubmittedAccessGrant(receipt)) return null;
  return (
    <AgentActionReceiptActions
      chatId={chatId}
      messageId={messageId}
      receipt={{
        kind: "access-grant",
        grant: receipt.grant,
        savedAt: receipt.submittedAt,
      }}
    />
  );
}
