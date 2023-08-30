<template>
  <DrawerContent :title="$t('settings.sensitive-data.grant-access')">
    <div class="divide-block-border space-y-8 w-[50rem] h-full">
      <SensitiveColumnTable
        :row-clickable="false"
        :show-operation="false"
        :row-selectable="false"
        :column-list="columnList"
      />

      <div class="w-full">
        <p class="mb-2">{{ $t("settings.sensitive-data.action.self") }}</p>
        <div class="flex space-x-5">
          <BBCheckbox :title="$t('settings.sensitive-data.action.query')" />
          <BBCheckbox :title="$t('settings.sensitive-data.action.export')" />
        </div>
      </div>

      <div class="w-full">
        <p class="mb-2">
          {{ $t("settings.sensitive-data.masking-level.self") }}
        </p>
        <div class="flex space-x-5">
          <label class="radio space-x-2">
            <input
              v-model="state.maskingLevel"
              :name="maskingLevelToJSON(MaskingLevel.PARTIAL)"
              type="radio"
              class="btn"
              :value="MaskingLevel.PARTIAL"
            />
            <span class="text-sm font-medium text-main whitespace-nowrap">{{
              $t("settings.sensitive-data.masking-level.partial")
            }}</span>
          </label>
          <label class="radio space-x-2">
            <input
              v-model="state.maskingLevel"
              :name="maskingLevelToJSON(MaskingLevel.NONE)"
              type="radio"
              class="btn"
              :value="MaskingLevel.NONE"
            />
            <span class="text-sm font-medium text-main whitespace-nowrap">{{
              $t("settings.sensitive-data.masking-level.none")
            }}</span>
          </label>
        </div>
      </div>

      <div class="w-full">
        <p class="mb-2">{{ $t("common.expiration") }}</p>
        <NDatePicker
          v-model:value="state.expirationTimestamp"
          style="width: 100%"
          type="datetime"
          :is-date-disabled="(date: number) => date < Date.now()"
          clearable
        />
        <span v-if="!state.expirationTimestamp" class="textinfolabel">{{
          $t("settings.sensitive-data.never-expires")
        }}</span>
      </div>

      <div class="w-full">
        <p class="mb-2">
          {{ $t("common.user") }}
        </p>
        <UserSelect
          v-model:users="state.userUidList"
          style="width: 100%"
          :multiple="true"
          :include-all="false"
        />
      </div>
    </div>

    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-3">
          <NButton @click.prevent="">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton :disabled="true" type="primary" @click.prevent="">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import {
  NButton,
  NDatePicker,
  NDrawer,
  NDrawerContent,
  NInput,
  NInputNumber,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { DrawerContent } from "@/components/v2";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import { SensitiveColumn } from "./types";

defineProps<{
  columnList: SensitiveColumn[];
}>();

interface LocalState {
  userUidList: string[];
  expirationTimestamp?: number;
  supportQuery: boolean;
  supportExport: boolean;
  maskingLevel: MaskingLevel;
}

const state = reactive<LocalState>({
  userUidList: [],
  supportQuery: false,
  supportExport: false,
  maskingLevel: MaskingLevel.PARTIAL,
});
</script>
