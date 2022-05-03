<template>
  <div class="space-y-4">
    <div class="textinfolabel mt-2">
      <i18n-t keypath="label.setting.description">
        <template #link>
          <a
            class="normal-link inline-flex items-center"
            href="https://bytebase.com/docs/features/tenant-database-management#labels"
            target="__BLANK"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4 ml-1" />
          </a>
        </template>
      </i18n-t>
    </div>
    <div>
      <BBTable
        class="mt-2"
        :column-list="COLUMN_LIST"
        :data-source="labelList"
        :show-header="true"
        :row-clickable="false"
      >
        <template #body="{ rowData: label }">
          <BBTableCell :left-padding="4" class="w-36 table-cell capitalize">
            <!-- will not capitalize label.key here due to it may be editable in the future -->
            <!-- capitalizing editable things may be confusing -->
            <!-- capitalizing only bb.prefixed things makes it inconsistent with others -->
            <!-- so just capitalize none of them -->
            {{ hidePrefix(label.key) }}
          </BBTableCell>
          <BBTableCell class="whitespace-nowrap">
            <div class="tags">
              <div v-for="(value, j) in label.valueList" :key="j" class="tag">
                <span>{{ value }}</span>
                <span
                  v-if="allowRemove"
                  class="remove"
                  @click="removeValue(label, value)"
                >
                  <heroicons-solid:x class="w-3 h-3" />
                </span>
              </div>
              <template v-if="allowEdit">
                <router-link
                  v-if="label.key === 'bb.environment'"
                  :to="{ name: 'workspace.environment' }"
                  class="h-6 px-1 py-1 inline-flex items-center rounded bg-white border border-control-border hover:bg-control-bg-hover cursor-pointer"
                >
                  {{ $t("common.manage") }}
                </router-link>
                <AddLabelValue
                  v-else
                  :label="label"
                  @add="(v) => addValue(label, v)"
                />
              </template>
            </div>
          </BBTableCell>
        </template>
      </BBTable>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { isDBAOrOwner, hidePrefix } from "../utils";
import { Label, LabelPatch } from "../types";
import { BBTableColumn } from "../bbkit/types";
import { BBTable, BBTableCell } from "../bbkit";
import { useI18n } from "vue-i18n";
import AddLabelValue from "../components/AddLabelValue.vue";
import { useCurrentUser, useLabelStore } from "@/store";
import { storeToRefs } from "pinia";

export default defineComponent({
  name: "SettingWorkspaceLabels",
  components: {
    BBTable,
    BBTableCell,
    AddLabelValue,
  },
  setup() {
    const { t } = useI18n();
    const labelStore = useLabelStore();
    const currentUser = useCurrentUser();

    const prepareLabelList = () => {
      labelStore.fetchLabelList();
    };

    watchEffect(prepareLabelList);

    const { labelList } = storeToRefs(labelStore);

    const allowEdit = computed(() => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowRemove = computed(() => {
      // no, we don't allow to remove values now
      return false;
      // return allowEdit.value;
    });

    const addValue = (label: Label, value: string) => {
      const labelPatch: LabelPatch = {
        valueList: [...label.valueList, value],
      };
      labelStore.patchLabel({
        id: label.id,
        labelPatch,
      });
    };

    const removeValue = (label: Label, value: string) => {
      const valueList = [...label.valueList];
      const index = valueList.indexOf(value);
      if (index < 0) return;
      valueList.splice(index, 1);
      const labelPatch: LabelPatch = {
        valueList,
      };
      labelStore.patchLabel({
        id: label.id,
        labelPatch,
      });
    };

    const COLUMN_LIST = computed((): BBTableColumn[] => [
      {
        title: t("setting.label.key"),
      },
      {
        title: t("setting.label.values"),
      },
    ]);

    return {
      COLUMN_LIST,
      labelList,
      hidePrefix,
      allowEdit,
      allowRemove,
      addValue,
      removeValue,
    };
  },
});
</script>

<style scoped lang="postcss">
.tags {
  @apply flex flex-wrap gap-2;
}
.tag {
  @apply h-6 bg-blue-100 border-blue-300 border px-2 rounded whitespace-nowrap inline-flex items-center;
}
.tag > .remove {
  @apply ml-1 -mr-1 p-px cursor-pointer hover:bg-blue-300 rounded-sm;
}
</style>
