<template>
  <NDropdown
    :show="showDropdown"
    :options="options"
    placement="bottom-end"
    @clickoutside="showDropdown = false"
    @select="handleSelect"
  >
    <NButton
      quaternary
      :size="isMdOrAbove?'medium':'small'"
      :class="
        orderBy
          ? 'text-accent! hover:text-accent'
          : 'text-control-placeholder hover:text-control'
      "
      @click="showDropdown = !showDropdown"
    >
      <template #icon>
        <ArrowUpDownIcon class="w-4 h-4" />
      </template>
      <span v-if="isMdOrAbove">{{ $t("issue.sort.sort") }}</span>
    </NButton>
  </NDropdown>
</template>

<script lang="tsx" setup>
import { ArrowUpDownIcon, CheckIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useWideScreen } from "@/composables/useWideScreen";

const props = defineProps<{
  orderBy: string;
}>();

const emit = defineEmits<{
  (event: "update:orderBy", value: string): void;
}>();

const { t } = useI18n();
const isMdOrAbove = useWideScreen();
const showDropdown = ref(false);

const checkIcon = (value: string) => {
  return () => (
    <CheckIcon
      class={`w-3 h-3 ${props.orderBy === value ? "text-accent" : "text-transparent"}`}
    />
  );
};

const options = computed((): DropdownOption[] => [
  {
    key: "create_time",
    label: t("issue.sort.created"),
    children: [
      {
        key: "create_time desc",
        label: t("issue.sort.descending"),
        icon: checkIcon("create_time desc"),
      },
      {
        key: "create_time asc",
        label: t("issue.sort.ascending"),
        icon: checkIcon("create_time asc"),
      },
    ],
  },
  {
    key: "update_time",
    label: t("issue.sort.updated"),
    children: [
      {
        key: "update_time desc",
        label: t("issue.sort.descending"),
        icon: checkIcon("update_time desc"),
      },
      {
        key: "update_time asc",
        label: t("issue.sort.ascending"),
        icon: checkIcon("update_time asc"),
      },
    ],
  },
]);

const handleSelect = (key: string) => {
  // Clicking the already-selected option clears back to default
  if (key === props.orderBy) {
    emit("update:orderBy", "");
  } else {
    emit("update:orderBy", key);
  }
  showDropdown.value = false;
};
</script>
