<template>
  <NSelect
    v-model:value="selectedRole"
    class="w-36!"
    size="small"
    :options="options"
    :placeholder="$t('instance.select-database-user')"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    :consistent-menu-width="false"
    :clearable="true"
    :disabled="!allowChange"
    :loading="loading"
  />
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { NSelect, type SelectOption } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import {
  updateSpecSheetWithStatement,
  usePlanContext,
} from "@/components/Plan/logic";
import { instanceRoleServiceClientConnect } from "@/connect";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import { setSheetStatement } from "@/utils";
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
const { selectedSpec } = useSelectedSpec();
const { sheetStatement, sheet, sheetReady } = useSpecSheet(selectedSpec);
const { plan, isCreating } = usePlanContext();

// Flag to prevent circular updates
const isUpdatingFromUI = ref(false);

const instanceRoles = ref<InstanceRole[]>([]);
const loading = ref(false);

const selectedRole = computed({
  get: () => contextSelectedRole.value,
  set: (value: string | undefined) => {
    isUpdatingFromUI.value = true;
    contextSelectedRole.value = value;
    setRoleInStatement(value);
    // Reset the flag after Vue's next tick to allow statement updates to propagate
    nextTick(() => {
      isUpdatingFromUI.value = false;
    });
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
  if (!sheetReady.value || !sheetStatement.value) {
    // If sheet is not ready or statement is empty, reset to undefined
    contextSelectedRole.value = undefined;
    return;
  }

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

// Watch for sheet statement changes to update role
// Only update when statement changes externally, not when we change it ourselves
watch(
  [sheetStatement, sheetReady],
  () => {
    if (!isUpdatingFromUI.value) {
      initializeFromStatement();
    }
  },
  { immediate: true }
);

const setRoleInStatement = async (roleName: string | undefined) => {
  const updatedStatement = updateRoleSetter(sheetStatement.value, roleName);

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
  }
  events.emit("update");
};
</script>
