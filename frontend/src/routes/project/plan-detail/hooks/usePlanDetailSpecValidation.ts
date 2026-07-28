import { useEffect, useState } from "react";
import { useAppStore } from "@/stores/app";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import { extractSheetUID, getSheetStatement } from "@/utils/v1/sheet";
import {
  getLocalSheetByName,
  useLocalSheetsVersion,
} from "../utils/localSheet";

const sameStringSet = (a: Set<string>, b: Set<string>): boolean => {
  if (a.size !== b.size) return false;
  for (const value of b) {
    if (!a.has(value)) return false;
  }
  return true;
};

const checkSpecStatement = async (spec: Plan_Spec): Promise<boolean> => {
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
      : await useAppStore.getState().getOrFetchSheetByName(sheetName);
    if (!sheet) {
      return true;
    }
    return getSheetStatement(sheet).trim() === "";
  } catch {
    return false;
  }
};

export function usePlanDetailSpecValidation(specs: Plan_Spec[]) {
  const [emptySpecIdSet, setEmptySpecIdSet] = useState<Set<string>>(
    () => new Set()
  );
  // Draft statements live in mutable local sheets outside React state; this
  // version bumps on every local-sheet write so edits re-run the validation
  // (the `specs` identity alone doesn't change while typing on a draft plan).
  const localSheetsVersion = useLocalSheetsVersion();

  useEffect(() => {
    let canceled = false;

    const validate = async () => {
      const next = new Set<string>();
      await Promise.all(
        specs.map(async (spec) => {
          if (await checkSpecStatement(spec)) {
            next.add(spec.id);
          }
        })
      );
      if (!canceled) {
        // Reuse the prior Set when membership is unchanged: this runs on every
        // keystroke (localSheetsVersion bumps per edit), and a fresh Set ref
        // each time would re-render the header for nothing.
        setEmptySpecIdSet((prev) => (sameStringSet(prev, next) ? prev : next));
      }
    };

    void validate();

    return () => {
      canceled = true;
    };
  }, [specs, localSheetsVersion]);

  return {
    emptySpecIdSet,
  };
}
