import { flatten, uniq } from "lodash-es";
import type {
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTreeState as TreeState,
} from "@/types";
import { formatEnvironmentName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Environment } from "@/types/v1/environment";
import type {
  SQLEditorSliceCreator,
  SQLEditorStoreState,
  TreeSlice,
} from "./types";

export const idForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  if (type === "instance" || type === "database") {
    return (target as Project | InstanceResource | Database).name;
  }
  if (type === "environment") {
    return formatEnvironmentName((target as Environment).id);
  }
  if (type === "label") {
    const kv = target as NodeTarget<"label">;
    return `labels/${kv.key}:${kv.value}`;
  }
  throw new Error(
    `should never reach this line, type=${type}, target=${target}`
  );
};

export const createTreeSlice: SQLEditorSliceCreator<TreeSlice> = (
  set,
  get
) => ({
  treeState: "UNSET",
  treeNodeKeysById: {},

  setTreeState: (state: TreeState) => set({ treeState: state }),

  collectTreeNode: (node) => {
    const { type, target } = node.meta;
    const id = idForSQLEditorTreeNodeTarget(type, target);
    set((s) => {
      const prev = s.treeNodeKeysById[id] ?? [];
      return {
        treeNodeKeysById: {
          ...s.treeNodeKeysById,
          [id]: [...prev, node.key],
        },
      };
    });
  },

  treeNodeKeysByTarget: (type, target) => {
    const id = idForSQLEditorTreeNodeTarget(type, target);
    return get().treeNodeKeysById[id] ?? [];
  },
});

export const selectAllTreeNodeKeys = (state: SQLEditorStoreState): string[] =>
  uniq(flatten(Object.values(state.treeNodeKeysById)));
