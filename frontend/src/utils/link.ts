export type ResourceType = "UNKNOWN" | "DATABASE";

const resourceToPrefix = new Map([
  ["DATABASE", "/db"],
  ["UNKNOWN", "/404"],
]);

export function linkfy(type: ResourceType, id: string): string {
  return [resourceToPrefix.get(type), id].join("/");
}
