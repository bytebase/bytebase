import { create } from "@bufbuild/protobuf";
import type { LabelListEditorItem } from "@/components/LabelListEditor";
import {
  type Database,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { MAX_LABEL_VALUE_LENGTH } from "@/utils/label";

export const buildDatabaseLabelsUpdate = (
  name: string,
  labels: Readonly<Record<string, string>>
) =>
  create(UpdateDatabaseRequestSchema, {
    database: { name, labels: { ...labels } },
    updateMask: { paths: ["labels"] },
  });
export const buildDatabaseEnvironmentUpdate = (
  name: string,
  environment: string
) =>
  create(UpdateDatabaseRequestSchema, {
    database: { name, environment },
    updateMask: { paths: ["environment"] },
  });

export const compareKeys = (a: string, b: string) =>
  a === b ? 0 : a < b ? -1 : 1;
export const labelsFingerprint = (labels: Record<string, string>) =>
  JSON.stringify(
    Object.keys(labels)
      .sort(compareKeys)
      .map((key) => [key, labels[key]])
  );
export const environmentFingerprint = (database: Database) =>
  JSON.stringify([
    database.environment ?? "",
    database.effectiveEnvironment ?? "",
    database.instanceResource?.environment ?? "",
  ]);
export const labelsChanged = (
  labels: Record<string, string>,
  rows: LabelListEditorItem[]
) =>
  rows.length !== Object.keys(labels).length ||
  new Set(rows.map(({ key }) => key)).size !== rows.length ||
  rows.some(
    ({ key, value }) => !Object.hasOwn(labels, key) || labels[key] !== value
  );
export const validRows = (rows: LabelListEditorItem[]) =>
  new Set(rows.map(({ key }) => key)).size === rows.length &&
  rows.every(
    ({ key, value, allowEmpty }) =>
      key && (value || allowEmpty) && value.length <= MAX_LABEL_VALUE_LENGTH
  );
export const labelMap = (rows: LabelListEditorItem[]): Record<string, string> =>
  Object.fromEntries(rows.map(({ key, value }) => [key, value]));
