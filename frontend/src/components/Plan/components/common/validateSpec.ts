import { ref, watchEffect } from "vue";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { useSheetV1Store } from "@/store";
import { extractSheetUID, getSheetStatement, sheetNameOfSpec } from "@/utils";
import { getLocalSheetByName } from "@/components/Plan";

// Hook to validate all specs in a plan
export const useSpecsValidation = (specs: Plan_Spec[]) => {
  const sheetStore = useSheetV1Store();
  const validationMap = ref(new Map<string, boolean>());

  const checkSpecStatement = async (spec: Plan_Spec): Promise<boolean> => {
    // Only changeDatabaseConfig and exportDataConfig specs have statements
    if (!spec.changeDatabaseConfig && !spec.exportDataConfig) {
      return false;
    }

    const sheetName = sheetNameOfSpec(spec);
    // If there's no sheet reference, it's definitely empty
    if (!sheetName) {
      return true;
    }

    try {
      // Get sheet content
      const uid = extractSheetUID(sheetName);
      const sheet = uid.startsWith("-")
        ? getLocalSheetByName(sheetName)
        : await sheetStore.getOrFetchSheetByName(sheetName);

      if (!sheet) {
        return true; // Consider empty if sheet not found
      }

      const statement = getSheetStatement(sheet);
      return !statement || statement.trim() === "";
    } catch {
      return false; // If error, don't show as empty
    }
  };

  watchEffect(async () => {
    const newMap = new Map<string, boolean>();
    
    // Check all specs
    await Promise.all(
      specs.map(async (spec) => {
        const isEmpty = await checkSpecStatement(spec);
        newMap.set(spec.id, isEmpty);
      })
    );
    
    validationMap.value = newMap;
  });

  const isSpecEmpty = (spec: Plan_Spec): boolean => {
    return validationMap.value.get(spec.id) ?? false;
  };

  return {
    isSpecEmpty,
  };
};