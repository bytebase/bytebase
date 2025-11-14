<template>
  <div v-bind="$attrs" :class="[!sticky && 'flex flex-col gap-y-8']">
    <div
      :class="[
        'px-0.5',
        sticky && 'pt-2 pb-4 sticky top-0 bg-white z-10',
        headerClass,
      ]"
    >
      <NSteps :current="currentIndex + 1" size="small" :status="'process'">
        <NStep
          v-for="(step, i) in stepList"
          :key="i"
          :title="step.title"
          :description="step.description"
        />
      </NSteps>
    </div>

    <div :class="[sticky && 'mb-8', paneClass]">
      <template v-for="(step, index) in stepList" :key="index">
        <slot
          v-if="currentIndex == index"
          :name="index"
          :next="
            () => {
              $emit('update:currentIndex', currentIndex + 1);
            }
          "
        />
      </template>
    </div>

    <div
      v-if="showFooter"
      class="pt-4 border-t border-block-border flex items-center gap-x-2 justify-between"
      :class="[sticky && 'pb-4 bg-white sticky bottom-0 z-10', footerClass]"
    >
      <div>
        <NButton v-if="showCancel" @click.prevent="$emit('cancel')">
          {{ cancelTitle }}
        </NButton>
      </div>

      <div class="flex items-center justify-between gap-x-2">
        <NButton
          v-if="currentIndex != 0"
          v-bind="backButtonProps"
          @click.prevent="$emit('update:currentIndex', currentIndex - 1)"
        >
          <heroicons-outline:chevron-left
            class="-ml-1 mr-1 h-5 w-5 text-control-light"
          />
          <span> {{ $t("common.back") }}</span>
        </NButton>
        <NButton
          v-if="currentIndex == stepList.length - 1"
          :disabled="!allowNext"
          type="primary"
          v-bind="finishButtonProps"
          @click.prevent="$emit('finish')"
        >
          {{ finishTitle }}
        </NButton>
        <NButton
          v-else-if="!stepList[currentIndex].hideNext"
          :disabled="!allowNext"
          type="primary"
          v-bind="nextButtonProps"
          @click.prevent="$emit('update:currentIndex', currentIndex + 1)"
        >
          {{ $t("common.next") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ButtonProps } from "naive-ui";
import { NButton, NStep, NSteps } from "naive-ui";
import { t } from "@/plugins/i18n";
import type { VueClass } from "@/utils/types";

withDefaults(
  defineProps<{
    currentIndex: number;
    showCancel?: boolean;
    allowNext?: boolean;
    showFooter?: boolean;
    sticky?: boolean;
    cancelTitle?: string;
    finishTitle?: string;
    paneClass?: VueClass;
    headerClass?: VueClass;
    footerClass?: VueClass;
    stepList: { title: string; description?: string; hideNext?: boolean }[];
    backButtonProps?: ButtonProps;
    nextButtonProps?: ButtonProps;
    finishButtonProps?: ButtonProps;
  }>(),
  {
    showCancel: true,
    allowNext: true,
    showFooter: true,
    sticky: false,
    cancelTitle: () => t("common.cancel"),
    finishTitle: () => t("common.finish"),
    paneClass: undefined,
    headerClass: undefined,
    footerClass: undefined,
    backButtonProps: undefined,
    nextButtonProps: undefined,
    finishButtonProps: undefined,
  }
);

defineEmits<{
  (event: "cancel"): void;
  (event: "finish"): void;
  (event: "update:currentIndex", step: number): void;
}>();
</script>
