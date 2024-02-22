<template>
  <div v-bind="$attrs" :class="[!sticky && 'space-y-8']">
    <div :class="['px-0.5', sticky && 'pt-2 pb-4 sticky top-0 bg-white z-10']">
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
      class="pt-4 border-t border-block-border flex items-center space-x-2 justify-between"
      :class="[sticky && 'pb-4 bg-white sticky bottom-0 z-10', footerClass]"
    >
      <div>
        <NButton v-if="showCancel" @click.prevent="$emit('cancel')">
          {{ $t("bbkit.common.cancel") }}
        </NButton>
      </div>

      <div class="flex items-center justify-between space-x-2">
        <NButton
          v-if="currentIndex != 0"
          v-bind="backButtonProps"
          @click.prevent="$emit('update:currentIndex', currentIndex - 1)"
        >
          <heroicons-outline:chevron-left
            class="-ml-1 mr-1 h-5 w-5 text-control-light"
          />
          <span> {{ $t("bbkit.common.back") }}</span>
        </NButton>
        <NButton
          v-if="currentIndex == stepList.length - 1"
          :disabled="!allowNext"
          type="primary"
          v-bind="finishButtonProps"
          @click.prevent="$emit('finish')"
        >
          {{ $t(finishTitle) }}
        </NButton>
        <NButton
          v-else-if="!stepList[currentIndex].hideNext"
          :disabled="!allowNext"
          type="primary"
          v-bind="nextButtonProps"
          @click.prevent="$emit('update:currentIndex', currentIndex + 1)"
        >
          {{ $t("bbkit.common.next") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSteps, NStep, NButton, ButtonProps } from "naive-ui";
import { VueClass } from "@/utils/types";

withDefaults(
  defineProps<{
    currentIndex: number;
    showCancel?: boolean;
    allowNext?: boolean;
    sticky?: boolean;
    finishTitle?: string;
    paneClass?: VueClass;
    footerClass?: VueClass;
    stepList: { title: string; description?: string; hideNext?: boolean }[];
    backButtonProps?: ButtonProps;
    nextButtonProps?: ButtonProps;
    finishButtonProps?: ButtonProps;
  }>(),
  {
    showCancel: true,
    allowNext: true,
    sticky: false,
    finishTitle: "bbkit.common.finish",
    paneClass: undefined,
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
