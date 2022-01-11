<template>
  <NPopover trigger="hover">
    <template #trigger>
      <div class="bb-database-matrix-item" @click="clickDatabase">
        <div class="sync-status whitespace-nowrap">
          <span class="tooltip-wrapper">
            <heroicons-solid:check
              v-if="database.syncStatus === 'OK'"
              class="w-4 h-4 text-success"
            />
            <heroicons-outline:exclamation
              v-else
              class="w-4 h-4 text-warning"
            />
          </span>
          <span class="flex-1">{{ database.syncStatus }}</span>
        </div>

        <div>{{ $t("common.version") }}: {{ database.schemaVersion }}</div>
      </div>
    </template>

    <div class="popover">
      <div class="instance flex items-center whitespace-pre-wrap">
        <InstanceEngineIcon :instance="database.instance" />
        <span class="flex-1">{{ instanceName(database.instance) }}</span>
      </div>

      <div class="last-sync flex items-center">
        {{ $t("db.last-successful-sync") }}
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </div>

      <div
        v-if="database.labels.length > 0"
        class="labels whitespace-nowrap flex-col items-start"
      >
        <div class="mb-2">{{ $t("common.labels") }}</div>
        <DatabaseLabels
          :labels="database.labels"
          :editable="false"
          class="flex-col items-start"
        />
      </div>
    </div>
  </NPopover>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { useRouter } from "vue-router";
import { Database } from "../../types";
import { databaseSlug } from "../../utils";
import InstanceEngineIcon from "../InstanceEngineIcon.vue";
import DatabaseLabels from "../DatabaseLabels";
import { NPopover } from "naive-ui";

export default defineComponent({
  name: "DatabaseMatrixItem",
  components: {
    InstanceEngineIcon,
    DatabaseLabels,
    NPopover,
  },
  props: {
    database: {
      type: Object as PropType<Database>,
      required: true,
    },
    clickable: {
      type: Boolean,
      default: true,
    },
    customClick: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["select-database"],
  setup(props, { emit }) {
    const router = useRouter();

    const clickDatabase = () => {
      const { clickable, customClick, database } = props;
      if (!clickable) return;

      if (customClick) {
        emit("select-database", database);
      } else {
        router.push(`/db/${databaseSlug(database)}`);
      }
    };

    return { clickDatabase };
  },
});
</script>

<style scoped lang="postcss">
.bb-database-matrix-item {
  @apply border-gray-300 border rounded px-2 py-0 divide-y cursor-pointer select-none hover:bg-gray-50;
}
.bb-database-matrix-item > * {
  @apply flex items-center py-1 gap-1;
}
.popover {
  @apply bg-white divide-y cursor-pointer select-none;
}
.popover > * {
  @apply py-1 gap-1;
}
</style>
