export const extractProjectWebhookID = (name: string) => {
  const pattern = /(?:^|\/)webhooks\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
