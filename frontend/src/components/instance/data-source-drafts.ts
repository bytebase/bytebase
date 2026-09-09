export type SourceDraftState = {
  instanceName: string | undefined;
  resetEvent: number;
};

export function invalidateSourceDrafts<T>(
  drafts: Map<string, Map<number, T>>,
  previous: SourceDraftState,
  next: SourceDraftState
): SourceDraftState {
  if (
    previous.instanceName !== next.instanceName ||
    previous.resetEvent !== next.resetEvent
  ) {
    drafts.clear();
  }
  return next;
}
