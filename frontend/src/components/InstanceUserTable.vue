<template>
  <BBTable
    :column-list="columnList"
    :data-source="instanceUserList"
    :show-header="true"
    :row-clickable="false"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #body="{ rowData: instanceUser }">
      <BBTableCell :left-padding="4" class="w-4">
        {{ instanceUser.name }}
      </BBTableCell>
      <BBTableCell class="whitespace-pre-wrap break-all">
        {{ instanceUser.grant.replaceAll("\n", "\n\n") }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBTable, BBTableCell } from "@/bbkit";
import type { InstanceUser } from "../types/InstanceUser";

export default {
  name: "InstanceUserTable",
  components: {
    BBTable,
    BBTableCell,
  },
  props: {
    instanceUserList: {
      required: true,
      type: Object as PropType<InstanceUser[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const columnList = computed(() => [
      {
        title: t("common.user"),
      },
      {
        title: t("instance.grants"),
      },
    ]);
    return {
      columnList,
    };
  },
};
</script>
