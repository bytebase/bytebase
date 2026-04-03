import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";

export type DataExportPrepStep = 1 | 2;

export interface DataExportPrepSeed {
  targetSelectState?: DatabaseSelectState;
  step?: DataExportPrepStep;
}

export const normalizeDataExportPrepSeed = (
  seed?: DataExportPrepSeed
): {
  targetSelectState?: DatabaseSelectState;
  step: DataExportPrepStep;
} => {
  const targetSelectState = seed?.targetSelectState
    ? {
        changeSource: seed.targetSelectState.changeSource,
        selectedDatabaseNameList: [
          ...seed.targetSelectState.selectedDatabaseNameList,
        ],
        selectedDatabaseGroup: seed.targetSelectState.selectedDatabaseGroup,
      }
    : undefined;

  return {
    targetSelectState,
    step: seed?.step ?? 1,
  };
};
