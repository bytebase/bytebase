<template>
  <div class="w-full space-y-4 text-sm">
    <div class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-end textlabel">
        {{ $t("common.version") }}
      </label>
      <div class="flex-1">
        <a
          :href="migrationHistoryLink"
          target="__BLANK"
          class="normal-link flex items-center gap-x-1"
        >
          {{ migrationHistory.version }}
        </a>
      </div>
    </div>

    <div v-if="migrationHistory.issueId" class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-end textlabel">
        {{ $t("common.issue") }}
      </label>
      <div class="flex-1">
        <a
          :href="`/issue/${migrationHistory.issueId}`"
          target="__BLANK"
          class="normal-link flex items-center gap-x-1"
        >
          {{ migrationHistory.issueId }}
        </a>
      </div>
    </div>

    <div class="flex items-start gap-x-4">
      <label class="w-1/4 flex justify-end textlabel">
        {{ $t("common.sql") }}
      </label>
      <div class="flex-1">
        <NPopover
          :disabled="migrationHistory.statement.length < MAX_SQL_LENGTH"
          style="max-height: 300px"
          placement="bottom"
          overlap
          width="trigger"
          scrollable
        >
          <highlight-code-block
            :code="migrationHistory.statement"
            class="whitespace-pre-wrap"
          />

          <template #trigger>
            <div class="w-full">
              {{ displayStatement }}
            </div>
          </template>
        </NPopover>
      </div>
    </div>

    <div class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-end textlabel">
        {{ $t("common.created-at") }}
      </label>
      <div class="flex-1">
        <span>{{ createdAt }}</span>
        <span class="textinfolabel ml-1">({{ timezone }})</span>
      </div>
    </div>

    <div class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-end textlabel">
        {{ $t("common.creator") }}
      </label>
      <div class="flex-1">{{ migrationHistory.creator }}</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import dayjs from "dayjs";
import { Database, MigrationHistory } from "@/types";
import { databaseSlug, migrationHistorySlug } from "@/utils";

const props = defineProps({
  database: {
    type: Object as PropType<Database>,
    required: true,
  },
  migrationHistory: {
    type: Object as PropType<MigrationHistory>,
    required: true,
  },
});

const MAX_SQL_LENGTH = 100;

const migrationHistoryLink = computed(() => {
  const { database, migrationHistory } = props;
  return `/db/${databaseSlug(database)}/history/${migrationHistorySlug(
    migrationHistory.id,
    migrationHistory.version
  )}`;
});

const createdAt = computed(() => {
  return dayjs(props.migrationHistory.createdTs * 1000).format(
    "YYYY-MM-DD HH:mm:ss"
  );
});
const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const displayStatement = computed((): string => {
  const { migrationHistory } = props;
  return migrationHistory.statement.length > MAX_SQL_LENGTH
    ? migrationHistory.statement.substring(0, MAX_SQL_LENGTH) + "..."
    : migrationHistory.statement;
});
</script>
