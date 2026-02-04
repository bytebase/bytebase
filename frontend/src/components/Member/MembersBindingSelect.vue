<template>
  <div class="w-full flex flex-col gap-y-2">
    <i18n-t
      v-if="projectName"
      keypath="settings.members.select-in-project"
      tag="span"
      class="textinfolabel"
    >
      <template #project>
        <span class="font-semibold">{{ projectName }}</span>
      </template>
    </i18n-t>
    <div
      class="w-full flex flex-col gap-y-2"
    >
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-account", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
        <NTooltip>
          <template #trigger>
            <CircleHelpIcon class="w-4 textinfolabel" />
          </template>
          <span>
            {{ $t("settings.members.select-account-hint") }}
          </span>
        </NTooltip>
      </div>
      <AccountSelect
        key="account-select"
        :value="memberList"
        :multiple="true"
        :disabled="disabled"
        :project-name="projectName"
        :include-all-users="includeAllUsers"
        @update:value="onMemberListUpdate($event as string[])"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { uniq } from "lodash-es";
import { CircleHelpIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { AccountSelect } from "@/components/v2";
import {
  extractGroupEmail,
  extractServiceAccountId,
  extractUserEmail,
  extractWorkloadIdentityId,
  groupNamePrefix,
  serviceAccountNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store";
import {
  getGroupEmailInBinding,
  getServiceAccountNameInBinding,
  getUserEmailInBinding,
  getWorkloadIdentityNameInBinding,
} from "@/types";
import { convertMemberToFullname } from "@/utils";

const props = withDefaults(
  defineProps<{
    // member binding list, could be
    // - user:{email}
    // - group:{email}
    // - serviceAccount:{email}
    // - workloadIdentity:{email}
    // We don't support mix group with other data.
    value: string[];
    required: boolean;
    projectName?: string;
    disabled?: boolean;
    allowChangeType?: boolean;
    includeAllUsers?: boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    allowChangeType: true,
    includeAllUsers: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", memberList: string[]): void;
}>();

const memberList = computed(() => {
  return props.value.map(convertMemberToFullname);
});

const convertFullnameToMember = (fullname: string) => {
  if (fullname.startsWith(groupNamePrefix)) {
    const email = extractGroupEmail(fullname);
    return getGroupEmailInBinding(email);
  } else if (fullname.startsWith(serviceAccountNamePrefix)) {
    const email = extractServiceAccountId(fullname);
    return getServiceAccountNameInBinding(email);
  } else if (fullname.startsWith(workloadIdentityNamePrefix)) {
    const email = extractWorkloadIdentityId(fullname);
    return getWorkloadIdentityNameInBinding(email);
  } else {
    const email = extractUserEmail(fullname);
    return getUserEmailInBinding(email);
  }
};

const onMemberListUpdate = (fullnameList: string[]) => {
  const memberListInBinding = uniq(fullnameList)
    .map(convertFullnameToMember)
    .filter((member) => !!member);

  emit("update:value", memberListInBinding);
};
</script>
