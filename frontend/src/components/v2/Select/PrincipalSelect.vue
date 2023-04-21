<template>
  <NSelect
    :value="principal"
    :options="options"
    :filterable="true"
    :filter="filterByName"
    :virtual-scroll="true"
    :render-label="renderLabel"
    :fallback-option="false"
    :placeholder="$t('principal.select')"
    class="bb-principal-select"
    style="width: 12rem"
    @update:value="$emit('update:principal', $event)"
  />
</template>

<script lang="ts" setup>
import { computed, watch, watchEffect, h } from "vue";
import { NSelect, SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";
import { uniqBy } from "lodash-es";
import UserIcon from "~icons/heroicons-outline/user";

import { useMemberStore, usePrincipalStore, useProjectStore } from "@/store";
import {
  Principal,
  PrincipalId,
  ProjectId,
  ProjectRoleType,
  RoleType,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  unknown,
} from "@/types";
import PrincipalAvatar from "@/components/PrincipalAvatar.vue";

interface PrincipalSelectOption extends SelectOption {
  value: PrincipalId;
  principal: Principal;
}

const props = withDefaults(
  defineProps<{
    principal: PrincipalId | undefined;
    project?: ProjectId;
    includeAll?: boolean;
    includeSystemBot?: boolean;
    includeServiceAccount?: boolean;
    includeArchived?: boolean;
    allowedRoleList?: RoleType[];
    allowedProjectMemberRoleList?: ProjectRoleType[];
    autoReset?: boolean;
    filter?: (principal: Principal, index: number) => boolean;
  }>(),
  {
    project: undefined,
    includeAll: false,
    includeSystemBot: false,
    includeServiceAccount: false,
    includeArchived: false,
    allowedRoleList: () => ["DEVELOPER", "DBA", "OWNER"],
    allowedProjectMemberRoleList: () => ["DEVELOPER", "OWNER"],
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:principal", value: PrincipalId | undefined): void;
}>();

const { t } = useI18n();
const projectStore = useProjectStore();
const memberStore = useMemberStore();
const principalStore = usePrincipalStore();

const prepare = () => {
  if (props.project && props.project !== UNKNOWN_ID) {
    projectStore.getOrFetchProjectById(props.project);
  } else {
    // Need not to fetch the entire member list since it's done in
    // root component
  }
};
watchEffect(prepare);

const getPrincipalListFromProject = (project: ProjectId) => {
  const principalList = projectStore
    .getProjectById(project)
    .memberList.filter((member) => {
      return props.allowedProjectMemberRoleList.includes(member.role);
    })
    .map((member) => member.principal);
  return uniqBy(principalList, (user) => user.id);
};
const getPrincipalListFromWorkspace = () => {
  return memberStore.memberList
    .filter((member) => {
      if (props.includeArchived) return true;
      return member.rowStatus === "NORMAL";
    })
    .filter((member) => {
      return props.allowedRoleList.includes(member.role);
    })
    .map((member) => member.principal);
};

const rawPrincipalList = computed(() => {
  const list =
    props.project && props.project !== UNKNOWN_ID
      ? getPrincipalListFromProject(props.project)
      : getPrincipalListFromWorkspace();

  return list.filter((principal) => {
    if (principal.type === "SERVICE_ACCOUNT" && !props.includeServiceAccount) {
      return false;
    }

    return true;
  });
});

const combinedPrincipalList = computed(() => {
  let list = [...rawPrincipalList.value];

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.principal === SYSTEM_BOT_ID || props.includeSystemBot) {
    list.unshift(principalStore.principalById(SYSTEM_BOT_ID));
  }
  if (props.principal === UNKNOWN_ID || props.includeAll) {
    const dummyAll = unknown("PRINCIPAL");
    dummyAll.name = t("common.all");
    list.unshift(dummyAll);
  }

  return list;
});

const renderAvatar = (principal: Principal) => {
  if (principal.id === UNKNOWN_ID) {
    return h(
      "div",
      {
        class:
          "bb-principal-select--avatar w-6 h-6 rounded-full border-2 border-current flex justify-center items-center select-none bg-white",
      },
      h(UserIcon, {
        class: "w-4 h-4 text-main text-current",
      })
    );
  } else {
    return h(PrincipalAvatar, {
      class: "bb-principal-select--avatar",
      principal,
      size: "SMALL",
    });
  }
};

const renderLabel = (option: SelectOption) => {
  const { principal } = option as PrincipalSelectOption;
  const avatar = renderAvatar(principal);
  const text = h("span", { class: "truncate" }, principal.name);
  return h(
    "div",
    {
      class: "flex items-center gap-x-2",
    },
    [avatar, text]
  );
};

const options = computed(() => {
  return combinedPrincipalList.value.map<PrincipalSelectOption>((principal) => {
    return {
      principal,
      value: principal.id,
      label: principal.name,
    };
  });
});

const filterByName = (pattern: string, option: SelectOption) => {
  const { principal } = option as PrincipalSelectOption;
  pattern = pattern.toLowerCase();
  return (
    principal.name.toLowerCase().includes(pattern) ||
    principal.email.includes(pattern.toLowerCase())
  );
};

// The user list might change if props change, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    props.principal &&
    !combinedPrincipalList.value.find((item) => item.id === props.principal)
  ) {
    emit("update:principal", undefined);
  }
};

watch([() => props.principal, combinedPrincipalList], resetInvalidSelection, {
  immediate: true,
});
</script>

<style>
.bb-principal-select .n-base-selection--active .bb-principal-select--avatar {
  opacity: 0.3;
}
</style>
