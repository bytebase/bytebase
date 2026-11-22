import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";

export function getPlanCreateAgentContextQuery(
  query: Record<string, unknown>,
  specs: Plan_Spec[],
  selectedSpecId: string
): Record<string, unknown> | undefined {
  const selectedSpec = specs.find((spec) => spec.id === selectedSpecId);
  const targets =
    selectedSpec?.config?.case === "changeDatabaseConfig"
      ? (selectedSpec.config.value.targets ?? [])
      : [];
  const databaseName =
    targets.length === 1 && isValidDatabaseName(targets[0]) ? targets[0] : "";
  if (query.agentDatabaseName === databaseName) return undefined;
  if (
    query.agentDatabaseName === undefined &&
    databaseName &&
    query.databaseList === databaseName
  ) {
    return undefined;
  }
  return { ...query, agentDatabaseName: databaseName };
}
