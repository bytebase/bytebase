import type { EditDataSource } from "./common";

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

export function deactivateExternalSecret(
  dataSource: EditDataSource,
  drafts: Map<
    string,
    Map<number, NonNullable<EditDataSource["externalSecret"]>>
  >
): EditDataSource {
  if (dataSource.externalSecret) {
    const sourceDrafts = drafts.get(dataSource.id) ?? new Map();
    sourceDrafts.set(
      dataSource.externalSecret.secretType,
      dataSource.externalSecret
    );
    drafts.set(dataSource.id, sourceDrafts);
  }
  return { ...dataSource, externalSecret: undefined };
}
