<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div
        v-for="(user, index) in state.userList"
        :key="index"
        class="grid grid-cols-2 gap-x-4 py-2 select-none"
      >
        <div>
          <div
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
            class="flex items-center"
          >
            <NInput
              v-model:value="user.email"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              class="!w-10 sm:!w-20 shrink-0"
              placeholder="foo"
              @blur="validateUser(user, index)"
              @input="clearValidationError(index)"
            />
            <span class="ml-1 textinfolabel">
              {{ serviceAccountEmailSuffix }}
            </span>
          </div>
          <NInput
            v-else
            v-model:value="user.email"
            :input-props="{ type: 'email', autocomplete: 'off' }"
            placeholder="foo@example.com"
            @blur="validateUser(user, index)"
            @input="clearValidationError(index)"
          />
          <p
            v-if="state.errorList[index]"
            id="email-error"
            class="mt-2 text-sm text-error"
          >
            {{ state.errorList[index] }}
          </p>
        </div>
        <div v-if="hasRBACFeature" class="sm:hidden w-36">
          <RoleSelect v-model:role="user.userRole" />
        </div>
        <div
          v-if="hasRBACFeature"
          class="hidden sm:flex sm:flex-row sm:items-center space-x-4"
          :class="state.errorList[index] ? '-mt-7' : ''"
        >
          <RoleRadioSelect v-model:role="user.userRole" />
        </div>
        <div
          v-if="canManageMember"
          class="col-span-2 flex justify-start gap-x-2 items-center text-sm text-gray-500 pt-2 ml-0.5"
        >
          <NSwitch
            :value="user.userType === UserType.SERVICE_ACCOUNT"
            size="small"
            @update:value="toggleUserServiceAccount(user, index, $event)"
          />
          <span>
            {{ $t("settings.members.create-as-service-account") }}
            <a
              target="_blank"
              href="https://www.bytebase.com/docs/get-started/terraform?source=console"
            >
              <heroicons-outline:question-mark-circle
                class="w-4 h-4 inline-block mb-0.5"
              />
            </a>
          </span>
        </div>
      </div>
    </div>

    <div class="flex justify-between">
      <span class="flex items-center">
        <NButton text @click="addUser">
          {{ $t("settings.members.add-more") }}
        </NButton>
      </span>

      <NButton
        type="primary"
        :disabled="!hasValidUserOnly()"
        @click="addOrInvite"
      >
        <template #icon>
          <heroicons-solid:plus v-if="canManageMember" class="h-5 w-5" />
          <heroicons-outline:mail v-else class="mr-2 h-5 w-5" />
        </template>
        {{ $t(`settings.members.${canManageMember ? "add" : "invites"}`) }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NInput, NSwitch } from "naive-ui";
import { computed, reactive } from "vue";
import { RoleSelect, RoleRadioSelect } from "@/components/v2";
import {
  useUIStateStore,
  featureToRef,
  useUserStore,
  useCurrentUserV1,
} from "@/store";
import { emptyUser } from "@/types";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { isValidEmail, hasWorkspacePermissionV1, randomString } from "@/utils";
import { copyServiceKeyToClipboardIfNeeded } from "./common";

interface LocalState {
  userList: User[];
  errorList: string[];
}

const serviceAccountEmailSuffix = "@service.bytebase.com";

const userStore = useUserStore();

const currentUserV1 = useCurrentUserV1();

const canManageMember = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-member",
    currentUserV1.value.userRole
  );
});

const hasRBACFeature = featureToRef("bb.feature.rbac");

const state = reactive<LocalState>({
  userList: [],
  errorList: [],
});

for (let i = 0; i < 3; i++) {
  state.userList.push(emptyUser());
  state.errorList.push("");
}

const toggleUserServiceAccount = (user: User, index: number, on: boolean) => {
  user.userType = on ? UserType.SERVICE_ACCOUNT : UserType.USER;
  validateUser(user, index);
};

const validateUserInternal = (user: User): string => {
  if (!user.email) {
    return "";
  }
  if (user.userType === UserType.SERVICE_ACCOUNT) {
    if (isValidEmail(user.email)) {
      return "Do not include @xxx suffix in service account";
    }
    const email = `${user.email}${serviceAccountEmailSuffix}`;
    const existed = userStore.getUserByEmail(email);
    if (existed) {
      return "Already a member";
    }
  } else {
    if (!isValidEmail(user.email)) {
      return "Invalid email address";
    } else {
      const existed = userStore.getUserByEmail(user.email);
      if (existed) {
        return "Already a member";
      }
    }
  }

  return "";
};

const validateUser = (user: User, index: number): boolean => {
  state.errorList[index] = validateUserInternal(user);
  return state.errorList[index].length == 0;
};

const clearValidationError = (index: number) => {
  state.errorList[index] = "";
};

const addUser = () => {
  state.userList.push(emptyUser());
  state.errorList.push("");
};

const hasValidUserOnly = () => {
  let hasEmailInput = false;
  let hasError = false;
  state.userList.forEach((user) => {
    if (user.email) {
      hasEmailInput = true;
      if (validateUserInternal(user).length > 0) {
        hasError = true;
        return;
      }
    }
  });
  return !hasError && hasEmailInput;
};

const addOrInvite = async () => {
  for (const user of state.userList) {
    const userCreate = cloneDeep(user);
    if (!userCreate.email) {
      continue;
    }
    if (
      userCreate.userType !== UserType.SERVICE_ACCOUNT &&
      !isValidEmail(userCreate.email)
    ) {
      continue;
    }

    // If admin feature is NOT enabled, then we set every user to OWNER role.
    if (!hasRBACFeature.value) {
      userCreate.userRole = UserRole.OWNER;
    }
    userCreate.title = userCreate.email.split("@")[0];
    if (userCreate.userType === UserType.SERVICE_ACCOUNT) {
      userCreate.email = `${userCreate.email}${serviceAccountEmailSuffix}`;
    }
    userCreate.password = randomString(20);

    const createdUser = await userStore.createUser(userCreate);
    copyServiceKeyToClipboardIfNeeded(createdUser);

    if (createdUser.userRole !== userCreate.userRole) {
      const userPatch = cloneDeep(createdUser);
      userPatch.userRole = userCreate.userRole;
      await userStore.updateUser({
        user: userPatch,
        updateMask: ["role"],
        regenerateRecoveryCodes: false,
        regenerateTempMfaSecret: false,
      });
    }

    useUIStateStore().saveIntroStateByKey({
      key: "member.addOrInvite",
      newState: true,
    });
  }
  state.userList.forEach((item) => {
    Object.assign(item, emptyUser());
  });
  state.errorList = [""];
};
</script>
