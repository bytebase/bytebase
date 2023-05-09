import { groupBy } from "lodash-es";
import { TransferOption, TreeOption } from "naive-ui";
import { useEnvironmentStore } from "@/store";
import { Database } from "@/types";

export interface DatabaseTreeOption<L = "environment" | "database">
  extends TreeOption {
  level: L;
  value: string;
}

export const mapTreeOptions = (databaseList: Database[]) => {
  const environmentStore = useEnvironmentStore();
  const databaseListGroupByEnvironment = groupBy(
    databaseList,
    (db) => db.instance.environment.id
  );
  return Object.keys(databaseListGroupByEnvironment).map<
    DatabaseTreeOption<"environment">
  >((environmentId) => {
    const environment = environmentStore.getEnvironmentById(environmentId);
    const group = databaseListGroupByEnvironment[environmentId];
    const children = group.map<DatabaseTreeOption<"database">>((db) => ({
      level: "database",
      value: `database-${db.id}`,
      label: db.name,
      isLeaf: true,
    }));
    return {
      level: "environment",
      value: `environment-${environmentId}`,
      label: environment.name,
      children,
    };
  });
};

export const flattenTreeOptions = (options: DatabaseTreeOption[]) => {
  return options.flatMap((option) => {
    return [
      option as TransferOption,
      ...((option.children as TransferOption[] | undefined) ?? []),
    ];
  });
};
