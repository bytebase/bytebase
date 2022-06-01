<template>
  <NPopover
    trigger="hover"
    :theme-overrides="{
      borderRadius: '0.375rem',
      padding: '0',
    }"
  >
    <template #trigger>
      <div class="flex items-center select-none space-x-1">
        <div class="sync-status">
          <heroicons-solid:check-circle
            v-if="database.syncStatus === 'OK'"
            class="p-1 w-8 h-8 text-success"
          />
          <heroicons-solid:exclamation
            v-else
            class="p-1 w-8 h-8 text-warning"
          />
        </div>
        <div class="flex flex-col items-start">
          <router-link
            :to="databaseDetailUrl"
            class="text-main whitespace-nowrap hover:underline"
          >
            {{ database.name }}
          </router-link>

          <router-link
            :to="schemaVersionUrl"
            class="text-sm text-control hover:underline"
          >
            {{ database.schemaVersion }}
          </router-link>
        </div>
      </div>
    </template>

    <div class="rounded-md bg-white divide-y">
      <div class="px-4 py-2 flex items-center whitespace-nowrap space-x-1">
        <span>
          <heroicons-solid:check
            v-if="database.syncStatus === 'OK'"
            class="w-4 h-4 text-success"
          />
          <heroicons-outline:exclamation v-else class="w-4 h-4 text-warning" />
        </span>
        <span class="flex-1">{{ database.syncStatus }}</span>
      </div>
      <div class="px-4 py-2 flex items-center whitespace-pre-wrap space-x-1">
        <InstanceEngineIcon :instance="database.instance" />
        <span class="flex-1">{{ instanceName(database.instance) }}</span>
      </div>

      <div class="px-4 py-2 flex items-center justify-between space-x-1">
        <span>{{ $t("db.last-successful-sync") }}</span>
        <span>{{ humanizeTs(database.lastSuccessfulSyncTs) }}</span>
      </div>

      <div
        v-if="displayLabelList.length > 0"
        class="px-4 py-2 labels whitespace-nowrap flex-col items-start"
      >
        <div
          v-for="label in displayLabelList"
          :key="label.key"
          class="flex items-center space-y-1"
        >
          <span class="capitalize text-left">
            {{ hidePrefix(label.key) }}
          </span>
          <span
            class="flex-1 h-px mx-2 border-b border-control-border border-dashed"
          ></span>
          <span class="text-right">{{ label.value }}</span>
        </div>
      </div>
    </div>
  </NPopover>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { Database, Label } from "../../types";
import { databaseSlug, isReservedDatabaseLabel, hidePrefix } from "../../utils";
import InstanceEngineIcon from "../InstanceEngineIcon.vue";
import { NPopover } from "naive-ui";

export default defineComponent({
  name: "DatabaseMatrixItem",
  components: {
    InstanceEngineIcon,
    NPopover,
  },
  props: {
    database: {
      type: Object as PropType<Database>,
      required: true,
    },
    labelList: {
      type: Array as PropType<Label[]>,
      default: () => [],
    },
  },
  setup(props) {
    const displayLabelList = computed(() => {
      return props.database.labels.filter((dbLabel) => {
        if (!dbLabel.value) return false;
        if (isReservedDatabaseLabel(dbLabel, props.labelList)) return false;
        return true;
      });
    });

    const databaseDetailUrl = computed((): string => {
      return `/db/${databaseSlug(props.database)}`;
    });

    const schemaVersionUrl = computed((): string => {
      return `/db/${databaseSlug(props.database)}#migration-history`;
    });

    return {
      databaseDetailUrl,
      schemaVersionUrl,
      displayLabelList,
      hidePrefix,
    };
  },
});
</script>
