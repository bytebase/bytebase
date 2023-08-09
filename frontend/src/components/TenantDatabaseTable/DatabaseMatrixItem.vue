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
            v-if="database.syncState === State.ACTIVE"
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
            {{ database.databaseName }}
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
            v-if="database.syncState === State.ACTIVE"
            class="w-4 h-4 text-success"
          />
          <heroicons-outline:exclamation v-else class="w-4 h-4 text-warning" />
        </span>
        <span class="flex-1">
          {{ database.syncState === State.ACTIVE ? "OK" : "NOT_FOUND" }}
        </span>
      </div>
      <div class="px-4 py-2 flex items-center whitespace-pre-wrap space-x-1">
        <InstanceV1Name :instance="database.instanceEntity" :link="false" />
      </div>

      <div class="px-4 py-2 flex items-center justify-between space-x-1">
        <span>{{ $t("db.last-successful-sync") }}</span>
        <span>{{ humanizeDate(database.successfulSyncTime) }}</span>
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

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import { computed, PropType } from "vue";
import { State } from "@/types/proto/v1/common";
import { ComposedDatabase } from "../../types";
import { databaseV1Slug, hidePrefix, PRESET_LABEL_KEYS } from "../../utils";
import { InstanceV1Name } from "../v2";

const props = defineProps({
  database: {
    type: Object as PropType<ComposedDatabase>,
    required: true,
  },
});

const displayLabelList = computed(() => {
  return Object.keys(props.database.labels)
    .map((key) => {
      const value = props.database.labels[key];
      return { key, value };
    })
    .filter((kv) => {
      if (!kv.value) return false;
      if (!PRESET_LABEL_KEYS.includes(kv.key)) return false;
      return true;
    });
});

const databaseDetailUrl = computed((): string => {
  return `/db/${databaseV1Slug(props.database)}`;
});

const schemaVersionUrl = computed((): string => {
  return `/db/${databaseV1Slug(props.database)}#change-history`;
});
</script>
