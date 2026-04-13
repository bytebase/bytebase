import { useEffect, useState } from "react";
import { useSheetV1Store } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import { extractSheetUID, getSheetStatement } from "@/utils/v1/sheet";
import { getLocalSheetByName } from "../utils/localSheet";

const checkSpecStatement = async (
  spec: Plan_Spec,
  sheetStore: ReturnType<typeof useSheetV1Store>
): Promise<boolean> => {
  if (
    spec.config?.case !== "changeDatabaseConfig" &&
    spec.config?.case !== "exportDataConfig"
  ) {
    return false;
  }

  if (
    spec.config?.case === "changeDatabaseConfig" &&
    spec.config.value.release
  ) {
    return false;
  }

  const sheetName = sheetNameOfSpec(spec);
  if (!sheetName) {
    return true;
  }

  try {
    const uid = extractSheetUID(sheetName);
    const sheet = uid.startsWith("-")
      ? getLocalSheetByName(sheetName)
      : await sheetStore.getOrFetchSheetByName(sheetName);
    if (!sheet) {
      return true;
    }
    return getSheetStatement(sheet).trim() === "";
  } catch {
    return false;
  }
};

export function useIssueDetailSpecValidation(specs: Plan_Spec[]) {
  const sheetStore = useSheetV1Store();
  const [emptySpecIdSet, setEmptySpecIdSet] = useState<Set<string>>(
    () => new Set()
  );

  useEffect(() => {
    let canceled = false;

    const validate = async () => {
      const next = new Set<string>();
      await Promise.all(
        specs.map(async (spec) => {
          if (await checkSpecStatement(spec, sheetStore)) {
            next.add(spec.id);
          }
        })
      );
      if (!canceled) {
        setEmptySpecIdSet(next);
      }
    };

    void validate();

    return () => {
      canceled = true;
    };
  }, [sheetStore, specs]);

  return {
    emptySpecIdSet,
    isSpecEmpty: (spec: Plan_Spec) => emptySpecIdSet.has(spec.id),
  };
}
