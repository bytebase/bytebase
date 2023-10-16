<template>
  <div class="bg-white flex flex-col gap-y-2 w-max p-2">
    <FactorItem
      v-for="(factor, i) in PRESET_FACTORS"
      :key="i"
      :factor="factor"
      @toggle="toggle(factor, $event)"
    />
    <template v-if="labelFactors.length > 0">
      <div class="text-control-placeholder font-medium">
        {{ $t("common.labels") }}
      </div>
      <FactorItem
        v-for="(factor, i) in labelFactors"
        :key="i"
        :factor="factor"
        @toggle="toggle(factor, $event)"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, uniq } from "lodash-es";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import { SQLEditorTreeFactor as Factor } from "@/types";
import { keyBy } from "@/utils";
import FactorItem from "./FactorItem.vue";

const treeStore = useSQLEditorTreeStore();
const { databaseList, factorList } = storeToRefs(treeStore);

const PRESET_FACTORS: Factor[] = ["project", "instance", "environment"];

const labelFactors = computed(() => {
  return uniq(databaseList.value.flatMap((db) => Object.keys(db.labels))).map(
    (key) => `label:${key}` as Factor
  );
});

const availableFactors = computed(() => {
  return [...PRESET_FACTORS, ...labelFactors.value];
});

const toggle = (factor: Factor, on: boolean) => {
  const checkedSet = keyBy(cloneDeep(factorList.value), (sf) => sf.factor);
  if (on) {
    checkedSet.set(factor, { factor, disabled: false });
  } else {
    checkedSet.delete(factor);
  }
  const updatedSet = availableFactors.value.filter((factor) =>
    checkedSet.has(factor)
  );

  const updatedList = updatedSet.map((factor) => checkedSet.get(factor)!);
  if (updatedList.every((sf) => sf.disabled)) {
    // When all left factors are disabled
    // Enforce the first one to enable
    updatedList[0].disabled = false;
  }
  factorList.value = updatedList;
  treeStore.buildTree();
};
</script>
