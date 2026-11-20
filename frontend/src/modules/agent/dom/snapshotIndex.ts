import type { IndexedElement } from "./domTree";

let currentEntries = new Map<string, IndexedElement>();

export function publishDomSnapshot(entries: readonly IndexedElement[]): void {
  currentEntries = new Map(entries.map((entry) => [entry.ref, entry]));
}

export function getElementByRef(ref: string): IndexedElement | undefined {
  return currentEntries.get(ref);
}
