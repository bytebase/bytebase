<template>
  <NSelect
    v-model:value="selectedRole"
    class="!w-40"
    consistent-menu-width
    size="small"
    :options="options"
    :placeholder="$t('instance.select-database-user')"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    :disabled="!allowChange"
    :loading="loading"
  />
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { NSelect, type SelectOption } from "naive-ui";
import { computed, ref, watch } from "vue";
import { instanceRoleServiceClientConnect } from "@/grpcweb";
import { DEFAULT_PAGE_SIZE } from "@/store/modules/common";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  parseStatement,
  updateRoleSetter,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { useInstanceRoleSettingContext } from "./context";

const {
  allowChange,
  selectedRole: contextSelectedRole,
  databases,
  events,
} = useInstanceRoleSettingContext();
const selectedSpec = useSelectedSpec();
const { sheetStatement, updateSheetStatement } = useSpecSheet(selectedSpec);

const instanceRoles = ref<InstanceRole[]>([]);
const loading = ref(false);

const selectedRole = computed({
  get: () => contextSelectedRole.value,
  set: (value: string | undefined) => {
    contextSelectedRole.value = value;
    if (value) {
      setRoleInStatement(value);
    }
  },
});

const database = computed(() => {
  // For instance roles, we just need to get from the first database
  // since all databases in a spec share the same instance
  return databases.value[0];
});

watch(
  () => database.value?.instance,
  async () => {
    if (!database.value) return;

    loading.value = true;
    try {
      const request = create(ListInstanceRolesRequestSchema, {
        parent: database.value.instance,
        pageSize: DEFAULT_PAGE_SIZE,
      });
      const response =
        await instanceRoleServiceClientConnect.listInstanceRoles(request);
      instanceRoles.value = response.roles;
    } finally {
      loading.value = false;
    }
  },
  {
    immediate: true,
  }
);

const options = computed(() => {
  return instanceRoles.value.map<SelectOption>((instanceRole) => {
    return {
      value: instanceRole.roleName,
      label: instanceRole.roleName,
    };
  });
});

// Initialize selected role from statement
const initializeFromStatement = () => {
  const parsed = parseStatement(sheetStatement.value);
  if (parsed.roleSetterBlock) {
    // Extract role name from the role setter block
    const match = parsed.roleSetterBlock.match(
      /SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/
    );
    if (match) {
      contextSelectedRole.value = match[1];
    } else {
      contextSelectedRole.value = undefined;
    }
  } else {
    contextSelectedRole.value = undefined;
  }
};

watch(
  () => selectedSpec.value?.id,
  () => {
    initializeFromStatement();
  },
  {
    immediate: true,
  }
);

const setRoleInStatement = (roleName: string) => {
  const updatedStatement = updateRoleSetter(sheetStatement.value, roleName);
  updateSheetStatement(updatedStatement);
  events.emit("update");
};
</script>
