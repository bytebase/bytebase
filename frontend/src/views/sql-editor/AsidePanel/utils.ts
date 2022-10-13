import { ConnectionAtom } from "@/types";
import type { useInstanceStore } from "@/store";

export const generateInstanceNode = (
  item: ConnectionAtom,
  instanceStore: ReturnType<typeof useInstanceStore>
) => {
  return {
    ...item,
    children: item.children?.map(generateDatabaseItem),
  };
};

export const generateDatabaseItem = (item: ConnectionAtom) => {
  return {
    ...item,
    children: item.children?.map(generateTableItem),
  };
};

export const generateTableItem = (item: ConnectionAtom) => {
  return {
    ...item,
    isLeaf: true,
  };
};
