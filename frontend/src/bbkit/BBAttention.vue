<template>
  <NAlert :title="displayTitle" :type="type">
    <div class="flex items-center justify-between">
      <slot name="default">
        <div v-if="description" class="mt-2 text-sm">
          <p class="whitespace-pre-wrap">
            {{ $te(description) ? $t(description) : description }}
            <LearnMoreLink v-if="link" :url="link" class="ml-1 text-sm" />
          </p>
        </div>
      </slot>

      <slot name="action">
        <div
          v-if="actionText != ''"
          class="flex items-center justify-end mt-2 md:mt-0 md:ml-2"
        >
          <NButton :type="buttonType" @click.prevent="$emit('click')">
            {{ $t(actionText) }}
          </NButton>
        </div>
      </slot>
    </div>
  </NAlert>
</template>

<script lang="ts" setup>
import { NAlert } from "naive-ui";
import { computed, withDefaults } from "vue";
import { useI18n } from "vue-i18n";

const props = withDefaults(
  defineProps<{
    type: "default" | "info" | "success" | "warning" | "error";
    title?: string;
    description?: string;
    link?: string;
    actionText?: string;
  }>(),
  {
    title: "bbkit.attention.default",
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
    default:
      return "primary";
  }
});
</script>
