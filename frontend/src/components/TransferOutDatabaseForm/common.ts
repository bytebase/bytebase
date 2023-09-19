import { groupBy } from "lodash-es";
import { TransferOption, TreeOption } from "naive-ui";
import { useEnvironmentV1Store } from "@/store";
import { ComposedDatabase } from "@/types";

export interface DatabaseTreeOption<L = "environment" | "database">
  extends TreeOption {
  level: L;
  value: string;
}

export const mapTreeOptions = (databaseList: ComposedDatabase[]) => {
  const environmentV1Store = useEnvironmentV1Store();
  const databaseListGroupByEnvironment = groupBy(
    databaseList,
    (db) => db.effectiveEnvironment
  );
  return Object.keys(databaseListGroupByEnvironment).map<
    DatabaseTreeOption<"environment">
  >((environmentName) => {
    const environment =
      environmentV1Store.getEnvironmentByName(environmentName);
    const group = databaseListGroupByEnvironment[environmentName];
    const children = group.map<DatabaseTreeOption<"database">>((db) => ({
      level: "database",
      value: `database-${db.uid}`,
      label: db.name,
      isLeaf: true,
    }));
    return {
      level: "environment",
      value: `environment-${environment?.uid}`,
      label: environment?.title,
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
