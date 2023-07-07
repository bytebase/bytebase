<template>
  <div class="w-full space-y-4 text-sm">
    <div class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-start textlabel">
        {{ $t("common.version") }}
      </label>
      <div class="flex-1">
        <a
          :href="link"
          target="__BLANK"
          class="normal-link flex items-center gap-x-1"
        >
          {{ changeHistory.version }}
        </a>
      </div>
    </div>

    <div v-if="changeHistory" class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-start textlabel">
        {{ $t("common.issue") }}
      </label>
      <div class="flex-1">
        <a
          :href="`/issue/${extractIssueUID(changeHistory.issue)}`"
          target="__BLANK"
          class="normal-link flex items-center gap-x-1"
        >
          {{ extractIssueUID(changeHistory.issue) }}
        </a>
      </div>
    </div>

    <div class="flex items-start gap-x-4">
      <label class="w-1/4 flex justify-start textlabel">
        {{ $t("common.sql") }}
      </label>
      <div class="flex-1">
        <NPopover
          :disabled="changeHistory.statement.length < MAX_SQL_LENGTH"
          style="max-height: 300px"
          placement="bottom"
          overlap
          width="trigger"
          scrollable
        >
          <highlight-code-block
            :code="changeHistory.statement"
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
      <label class="w-1/4 flex justify-start textlabel">
        {{ $t("common.created-at") }}
      </label>
      <div class="flex-1">
        <span>{{ createdAt }}</span>
        <span class="textinfolabel ml-1">({{ timezone }})</span>
      </div>
    </div>

    <div class="flex items-center gap-x-4">
      <label class="w-1/4 flex justify-start textlabel">
        {{ $t("common.creator") }}
      </label>
      <div v-if="creator" class="flex-1">
        {{ creator.title }} ({{ creator.email }})
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import dayjs from "dayjs";
import {
  extractUserResourceName,
  extractIssueUID,
  changeHistoryLink as makeChangeHistoryLink,
} from "@/utils";
import { ComposedDatabase } from "@/types";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { useUserStore } from "@/store";

const props = defineProps({
  database: {
    type: Object as PropType<ComposedDatabase>,
    required: true,
  },
  changeHistory: {
    type: Object as PropType<ChangeHistory>,
    required: true,
  },
});

const MAX_SQL_LENGTH = 100;

const link = computed(() => {
  return makeChangeHistoryLink(props.changeHistory);
});

const createdAt = computed(() => {
  const date = props.changeHistory.createTime ?? new Date(0);
  return dayjs(date).format("YYYY-MM-DD HH:mm:ss");
});
const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const displayStatement = computed((): string => {
  const { changeHistory } = props;
  return changeHistory.statement.length > MAX_SQL_LENGTH
    ? changeHistory.statement.substring(0, MAX_SQL_LENGTH) + "..."
    : changeHistory.statement;
});

const creator = computed(() => {
  const email = extractUserResourceName(props.changeHistory.creator);
  return useUserStore().getUserByEmail(email);
});
</script>
