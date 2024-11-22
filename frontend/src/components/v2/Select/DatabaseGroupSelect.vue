<template>
  <NSelect
    :value="selected"
    :options="dbGroupOptions"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="'Select Database Group'"
    @update:value="$emit('update:selected', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useDBGroupListByProject } from "@/store";

const props = withDefaults(
  defineProps<{
    project: string;
    selected?: string;
    disabled?: boolean;
    clearable?: boolean;
  }>(),
  {
    clearable: false,
    selected: undefined,
  }
);

defineEmits<{
  (event: "update:selected", name: string | undefined): void;
}>();

const { dbGroupList } = useDBGroupListByProject(props.project);

const dbGroupOptions = computed(() => {
  return dbGroupList.value.map<SelectOption>((dbGroup) => ({
    value: dbGroup.name,
    label: dbGroup.databaseGroupName,
  }));
});
</script>
