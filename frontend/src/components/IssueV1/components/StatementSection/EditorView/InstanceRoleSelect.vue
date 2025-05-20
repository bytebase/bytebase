<template>
  <div class="w-full flex flex-row justify-start items-center gap-2">
    <span class="shrink-0">{{ $t("common.role.self") }}</span>
    <NSelect
      v-model:value="state.selectedRole"
      class="!w-40 grow"
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
import { NSelect, type SelectOption } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { instanceRoleServiceClient } from "@/grpcweb";
import { DEFAULT_PAGE_SIZE } from "@/store/modules/common";
import type { InstanceRole } from "@/types/proto/v1/instance_role_service";
import { useEditorContext } from "./context";

/**
 * Regular expression to match and capture the role name in a specific comment format.
 * The expected format is:
 * /* === Bytebase Role Setter. DO NOT EDIT. === *\/
 * SET ROLE <role_name>;
 *
 * The regex captures the role name (\w+) following the "SET ROLE" statement.
 */
const ROLE_SETTER_REGEX =
  /\/\*\s*=== Bytebase Role Setter\. DO NOT EDIT\. === \*\/\s*SET ROLE (\w+);/;

interface LocalState {
  selectedRole?: string;
}

const editorContext = useEditorContext();

const { issue, selectedTask } = useIssueContext();
const state = reactive<LocalState>({});

const database = computed(() => {
  return databaseForTask(issue.value.projectEntity, selectedTask.value);
});

const instanceRoles = ref<InstanceRole[]>([]);

watch(
  () => database.value.instance,
  async () => {
    const { roles } = await instanceRoleServiceClient.listInstanceRoles({
      parent: database.value.instance,
      pageSize: DEFAULT_PAGE_SIZE,
    });
    instanceRoles.value = roles;
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
    // Initialize selected role from statement using regex.
    const match = editorContext.statement.value.match(ROLE_SETTER_REGEX);
    if (match) {
      state.selectedRole = match[1];
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
  const roleSetterTemplate = `/* === Bytebase Role Setter. DO NOT EDIT. === */\nSET ROLE ${roleName};`;
  let statement = "";
  if (ROLE_SETTER_REGEX.test(editorContext.statement.value)) {
    statement = editorContext.statement.value.replace(
      ROLE_SETTER_REGEX,
      roleSetterTemplate
    );
  } else {
    statement = roleSetterTemplate + "\n" + editorContext.statement.value;
  }
  editorContext.setStatement(statement);
};
</script>
