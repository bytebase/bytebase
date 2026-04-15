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

const isSameSet = (prev: Set<string>, next: Set<string>) => {
  if (prev.size !== next.size) {
    return false;
  }
  for (const value of next) {
    if (!prev.has(value)) {
      return false;
    }
  }
  return true;
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
        setEmptySpecIdSet((prev) => (isSameSet(prev, next) ? prev : next));
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
