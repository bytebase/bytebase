import { h } from "vue";
import { ConnectionAtom } from "@/types";
import type { useInstanceStore } from "@/store";
import InstanceEngineIconVue from "@/components/InstanceEngineIcon.vue";
import HeroiconsOutlineDatabase from "~icons/heroicons-outline/database";
import HeroiconsOutlineTable from "~icons/heroicons-outline/table";

export const generateInstanceNode = (
  item: ConnectionAtom,
  instanceStore: ReturnType<typeof useInstanceStore>
) => {
  const instance = instanceStore.getInstanceById(item.id);
  return {
    ...item,
    prefix: () =>
      h(InstanceEngineIconVue, {
        instance,
      }),
    children: item.children?.map(generateDatabaseItem),
  };
};

export const generateDatabaseItem = (item: ConnectionAtom) => {
  return {
    ...item,
    prefix: () =>
      h(HeroiconsOutlineDatabase, {
        class: "h-4 w-4",
      }),
    children: item.children?.map(generateTableItem),
  };
};

export const generateTableItem = (item: ConnectionAtom) => {
  return {
    ...item,
    isLeaf: true,
    prefix: () =>
      h(HeroiconsOutlineTable, {
        class: "h-4 w-4",
      }),
  };
};
