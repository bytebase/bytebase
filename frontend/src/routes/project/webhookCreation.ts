import type { Webhook } from "@/types/proto-es/v1/project_service_pb";

// findAddedWebhook picks out the webhook a create call added, so the form can
// navigate to it, and returns undefined when it cannot tell.
//
// AddWebhook answers with the project rather than the webhook it created, so
// the name the server generated has to be inferred from the response. Matching
// on the url used to do it and no longer can: reads do not return a webhook's
// url, because that url is the credential for posting into the customer's chat.
// What is left is a name absent from `before`, plus the title and type
// submitted. Neither half identifies it alone — somebody else creating a
// webhook while this form was open also has a name this client has not seen.
//
// Together they still do not identify it uniquely. Two clients creating
// webhooks with the same title and the same type are indistinguishable from
// here, and the response's ordering is not specified, so picking either one
// would be a guess. It is not a harmless guess: the caller navigates to the
// webhook's edit page, where a delete lands on whichever webhook was picked.
// So an ambiguous answer is no answer, and the caller sends the user to the
// list to choose. Identifying it properly needs the create RPC to return the
// name it generated (BOT-80).
export const findAddedWebhook = (
  before: Webhook[],
  after: Webhook[],
  created: Pick<Webhook, "title" | "type">
): Webhook | undefined => {
  const known = new Set(before.map((webhook) => webhook.name));
  const candidates = after.filter(
    (webhook) =>
      !known.has(webhook.name) &&
      webhook.title === created.title &&
      webhook.type === created.type
  );
  return candidates.length === 1 ? candidates[0] : undefined;
};
