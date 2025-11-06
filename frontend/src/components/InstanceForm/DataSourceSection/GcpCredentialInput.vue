<template>
  <div class="flex flex-col gap-y-1 w-full">
    <p class="textinfolabel">
      <span>{{ $t("instance.create-gcp-credentials") }}</span>
      <a
        href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
        target="_blank"
        class="normal-link inline-flex items-center ml-1"
      >
        <span>{{ $t("common.detailed-guide") }}</span>
        <heroicons-outline:external-link class="w-4 h-4 ml-1" />
      </a>
    </p>
    <DroppableTextarea
      :value="value"
      :resizable="false"
      :placeholder="$t('instance.type-or-paste-credentials-write-only')"
      class="w-full h-24 whitespace-pre-wrap"
      @update:value="$emit('update:value', $event ?? '')"
    />
  </div>
</template>

<script lang="ts" setup>
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";

defineProps<{
  value: string | undefined;
}>();

defineEmits<{
  (name: "update:value", value: string): void;
}>();
</script>
