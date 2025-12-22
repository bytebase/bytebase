import { useSheetV1Store, useStorageStore } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { extractSheetUID, getSheetStatement } from "@/utils";
import { sheetNameForSpec } from "../plan";
import { getLocalSheetByName } from "../sheet";
import type { InitialSQL } from "./create";

export const extractInitialSQLFromQuery = async (
  query: Record<string, string>
): Promise<InitialSQL> => {
  const storageStore = useStorageStore();
  const sql = query.sql;
  if (sql && typeof sql === "string") {
    return {
      sql,
    };
  }
  const sqlStorageKey = query.sqlStorageKey;
  if (sqlStorageKey && typeof sqlStorageKey === "string") {
    const sql = (await storageStore.get<string>(sqlStorageKey)) || "";
    return {
      sql,
    };
  }
  const sqlMapStorageKey = query.sqlMapStorageKey;
  if (sqlMapStorageKey && typeof sqlMapStorageKey === "string") {
    const sqlMap =
      (await storageStore.get<Record<string, string>>(sqlMapStorageKey)) || {};
    return { sqlMap };
  }
  return {};
};

export const isValidSpec = (spec: Plan_Spec): boolean => {
  if (spec.config?.case === "changeDatabaseConfig") {
    const sheetName = sheetNameForSpec(spec);
    if (!sheetName) {
      return false;
    }
    const uid = extractSheetUID(sheetName);
    if (uid.startsWith("-")) {
      const sheet = getLocalSheetByName(sheetName);
      if (getSheetStatement(sheet).length === 0) {
        return false;
      }
    } else {
      const sheet = useSheetV1Store().getSheetByName(sheetName);
      if (!sheet) {
        return false;
      }
      if (getSheetStatement(sheet).length === 0) {
        return false;
      }
    }
  }
  return true;
};
