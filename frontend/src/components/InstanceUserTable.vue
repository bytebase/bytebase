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
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { InstanceUser } from "../types/InstanceUser";

export default {
  name: "InstanceUserTable",
  components: {},
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
        title: t("common.User"),
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
