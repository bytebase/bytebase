// Shared list of dynamic key prefixes used by both extraction and lint scripts.
// Keys under these prefixes are constructed at runtime via template literals.
export const DYNAMIC_PREFIXES = [
  "dynamic.subscription.features.",
  "dynamic.subscription.purchase.features.",
  "dynamic.settings.sensitive-data.semantic-types.template.",
  "subscription.plan.",
  "settings.sensitive-data.algorithms.",
];
