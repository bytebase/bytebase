<template>
  <NSelect
    v-bind="$attrs"
    :value="role"
    :options="options"
    :placeholder="$t('instance.select-database-user')"
    :filter="filterByTitle"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    @update:value="onRoleChange"
  />
</template>

<script lang="ts" setup>
import { NSelect, type SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import { useInstanceV1Store } from "@/store";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";

interface InstanceRoleSelectOption extends SelectOption {
  value: string;
  instanceRole: InstanceRole;
}

const props = defineProps<{
  instanceName: string;
  filter?: (role: InstanceRole) => boolean;
  role?: string;
}>();

const emit = defineEmits<{
  (event: "update:instance-role", role: InstanceRole | undefined): void;
}>();

const instanceV1Store = useInstanceV1Store();

const instanceRoles = computed(
  () => instanceV1Store.getInstanceByName(props.instanceName)?.roles
);

watch(
  () => props.instanceName,
  async (instanceName) => {
    if (instanceName) {
      await instanceV1Store.getOrFetchInstanceByName(instanceName);
      emit("update:instance-role", undefined);
    }
  },
  { immediate: true }
);

const options = computed(() => {
  return filteredInstanceRoleList.value.map<InstanceRoleSelectOption>(
    (instanceRole) => {
      return {
        instanceRole,
        value: instanceRole.name,
        label: instanceRole.roleName,
      };
    }
  );
});

const filteredInstanceRoleList = computed(() => {
  if (!props.filter) return instanceRoles.value;
  return instanceRoles.value.filter(props.filter);
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { instanceRole } = option as InstanceRoleSelectOption;
  return instanceRole.roleName.toLowerCase().includes(pattern.toLowerCase());
};

const onRoleChange = (roleName: string | undefined) => {
  const role = instanceRoles.value.find((role) => role.name === roleName);
  emit("update:instance-role", role);
};
</script>
