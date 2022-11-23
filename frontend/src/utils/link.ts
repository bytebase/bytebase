import { ResourceType } from "../types";

const resourceToPrefix = new Map([["DATABASE", "/db"]]);

export function linkfy(type: ResourceType, id: string): string {
  return [resourceToPrefix.get(type), id].join("/");
}

export function absolutifyLink(rel: string): string {
  const anchor = document.createElement("a");
  anchor.setAttribute("href", rel);
  return anchor.href;
}
