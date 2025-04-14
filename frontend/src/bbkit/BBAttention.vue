<template>
  <NAlert :title="displayTitle" :type="type">
    <div
      class="flex flex-col justify-start items-start md:flex-row md:items-center md:justify-between"
    >
      <slot name="default">
        <div v-if="description" class="text-sm">
          <p class="whitespace-pre-wrap">
            {{ $te(description) ? $t(description) : description }}
            <LearnMoreLink v-if="link" :url="link" class="ml-1 text-sm" />
          </p>
        </div>
      </slot>

      <slot name="action">
        <div
          v-if="actionText != ''"
          class="flex items-center justify-end mt-4 md:mt-0 md:ml-2"
        >
          <NButton
            :type="buttonType"
            size="small"
            @click.prevent="$emit('click')"
          >
            {{ $t(actionText) }}
          </NButton>
        </div>
      </slot>
    </div>
  </NAlert>
</template>

<script lang="ts" setup>
import { NAlert, NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";

const props = withDefaults(
  defineProps<{
    type: "default" | "info" | "success" | "warning" | "error";
    title?: string;
    description?: string;
    link?: string;
    actionText?: string;
  }>(),
  {
    title: "",
    description: "",
    link: undefined,
    actionText: "",
  }
);

defineEmits<{
  (event: "click"): void;
}>();

const { t, te } = useI18n();

const displayTitle = computed(() => {
  const { title } = props;
  if (te(title)) return t(title);
  return title;
});

const buttonType = computed(() => {
  switch (props.type) {
    case "error":
      return "default";
    case "warning":
      return "default";
    default:
      return "primary";
  }
});
</script>
