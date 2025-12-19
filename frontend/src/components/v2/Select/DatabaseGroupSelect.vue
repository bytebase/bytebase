<template>
  <LocalResourceSelector
    v-bind="$attrs"
    :placeholder="$t('database-group.select')"
    :value="selected"
    :options="dbGroupOptions"
    :disabled="disabled"
    @update:value="$emit('update:selected', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useDBGroupListByProject } from "@/store";
import LocalResourceSelector from "./LocalResourceSelector.vue";

const props = withDefaults(
  defineProps<{
    project: string;
    selected?: string;
    disabled?: boolean;
  }>(),
  {
    selected: undefined,
  }
);

defineEmits<{
  (event: "update:selected", name: string | undefined): void;
}>();

const { dbGroupList } = useDBGroupListByProject(props.project);

const dbGroupOptions = computed(() => {
  return dbGroupList.value.map((dbGroup) => ({
    resource: dbGroup,
    value: dbGroup.name,
    label: dbGroup.title,
  }));
});
</script>
