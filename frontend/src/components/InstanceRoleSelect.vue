<template>
  <BBSelect
    :selected-item="selectedInstanceRole"
    :item-list="filteredInstanceUserList"
    :placeholder="$t('instance.select-database-user')"
    :show-prefix-item="true"
    @select-item="handleSelectItem"
  >
    <template #menuItem="{ item: instance }: { item: InstanceRole }">
      {{ instance.roleName }}
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { PropType, computed, ref, watch } from "vue";
import { useInstanceV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { InstanceRole } from "@/types/proto/v1/instance_role_service";

const props = defineProps({
  role: {
    type: String,
    default: undefined,
  },
  instanceId: {
    type: String,
    default: String(UNKNOWN_ID),
  },
  filter: {
    type: Function as PropType<(role: InstanceRole) => boolean>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (event: "select", role: string | undefined): void;
}>();

const instanceV1Store = useInstanceV1Store();
const selectedInstanceRole = ref<InstanceRole>();
const instanceRoleList = ref<InstanceRole[]>([]);

watch(
  () => props.instanceId,
  async () => {
    selectedInstanceRole.value = undefined;
    const instance = instanceV1Store.getInstanceByUID(props.instanceId);
    instanceRoleList.value = await instanceV1Store.fetchInstanceRoleListByName(
      instance.name
    );
    emit("select", undefined);
  },
  { immediate: true }
);

watch(
  () => props.role,
  () => {
    selectedInstanceRole.value = instanceRoleList.value.find(
      (user) => user.name === props.role
    );
  }
);

const filteredInstanceUserList = computed(() => {
  if (!props.filter) return instanceRoleList.value;
  return instanceRoleList.value.filter(props.filter);
});

const handleSelectItem = (instanceRole: InstanceRole) => {
  emit("select", instanceRole.name);
};
</script>
