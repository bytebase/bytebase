<template>
  <div class="w-full flex flex-row justify-start items-center gap-2">
    <span class="shrink-0">{{ $t("common.role.self") }}</span>
    <NSelect
      v-model:value="state.selectedRole"
      class="w-40! grow"
      consistent-menu-width
      size="small"
      :options="options"
      :placeholder="$t('instance.select-database-user')"
      :filterable="true"
      :virtual-scroll="true"
      :fallback-option="false"
    />
  </div>
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { NSelect, type SelectOption } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { instanceRoleServiceClientConnect } from "@/connect";
import { useCurrentProjectV1 } from "@/store";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import { databaseForTask } from "@/utils";
import { useEditorContext } from "./context";
import { parseStatement, updateRoleSetter } from "./directiveUtils";

interface LocalState {
  selectedRole?: string;
}

const editorContext = useEditorContext();

const { selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();
const state = reactive<LocalState>({});

const database = computed(() => {
  return databaseForTask(project.value, selectedTask.value);
});

const instanceRoles = ref<InstanceRole[]>([]);

watch(
  () => database.value.instance,
  async () => {
    const request = create(ListInstanceRolesRequestSchema, {
      parent: database.value.instance,
    });
    const response =
      await instanceRoleServiceClientConnect.listInstanceRoles(request);
    instanceRoles.value = response.roles;
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

watch(
  () => selectedTask.value.name,
  async () => {
    // Initialize selected role from statement
    const parsed = parseStatement(editorContext.statement.value);
    if (parsed.roleSetterBlock) {
      // Extract role name from the role setter block
      const match = parsed.roleSetterBlock.match(
        /SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/
      );
      if (match) {
        state.selectedRole = match[1];
      } else {
        state.selectedRole = undefined;
      }
    } else {
      state.selectedRole = undefined;
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => state.selectedRole,
  async () => {
    if (state.selectedRole) {
      setRoleInTaskStatement(state.selectedRole);
    }
  }
);

const setRoleInTaskStatement = (roleName: string) => {
  const updatedStatement = updateRoleSetter(
    editorContext.statement.value,
    roleName
  );
  editorContext.setStatement(updatedStatement);
};
</script>
