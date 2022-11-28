<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div
        v-for="(user, index) in state.userList"
        :key="index"
        class="flex flex-col py-2 select-none"
      >
        <div class="flex justify-between">
          <p id="add_or_invite_members_helper" class="sr-only">
            {{ $t("settings.members.helper") }}
          </p>
          <div class="flex flex-row space-x-4">
            <div class="flex-grow">
              <div v-if="user.isServiceAccount" class="flex items-center">
                <input
                  v-model="user.email"
                  type="text"
                  name="add_or_invite_members"
                  autocomplete="off"
                  class="w-10 sm:w-20 textfield lowercase"
                  placeholder="foo"
                  aria-describedby="add_or_invite_members_helper"
                  @blur="validateUser(user, index)"
                  @input="clearValidationError(index)"
                />
                <span class="ml-1">@service.bytebase.com</span>
              </div>
              <input
                v-else
                v-model="user.email"
                type="email"
                name="add_or_invite_members"
                autocomplete="off"
                class="w-36 sm:w-64 textfield lowercase"
                placeholder="foo@example.com"
                aria-describedby="add_or_invite_members_helper"
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
              <RoleSelect
                :selected-role="user.role"
                @change-role="
                  (role) => {
                    user.role = role;
                  }
                "
              />
            </div>
            <div
              v-if="hasRBACFeature"
              class="hidden sm:flex sm:flex-row space-x-4"
              :class="state.errorList[index] ? '-mt-7' : ''"
            >
              <div class="radio">
                <input
                  v-model="user.role"
                  :name="`add_or_invite_role${index}`"
                  tabindex="-1"
                  type="radio"
                  class="btn"
                  value="OWNER"
                />
                <label class="label">{{ $t("common.role.owner") }}</label>
              </div>
              <div class="radio">
                <input
                  v-model="user.role"
                  :name="`add_or_invite_role${index}`"
                  tabindex="-1"
                  type="radio"
                  class="btn"
                  value="DBA"
                />
                <label class="label">{{ $t("common.role.dba") }}</label>
              </div>
              <div class="radio">
                <input
                  v-model="user.role"
                  :name="`add_or_invite_role${index}`"
                  tabindex="-1"
                  type="radio"
                  class="btn"
                  value="DEVELOPER"
                />
                <label class="label">{{ $t("common.role.developer") }}</label>
              </div>
            </div>
          </div>
        </div>
        <div
          v-if="canManageMember"
          class="flex justify-start gap-x-2 items-center text-sm text-gray-500 pt-2 ml-0.5"
        >
          <NSwitch
            v-model:value="user.isServiceAccount"
            size="small"
            @change="validateUser(user, index)"
          />
          <span>
            {{ $t("settings.members.create-as-service-account") }}
            <a
              target="_blank"
              href="https://www.bytebase.com/zh/docs/get-started/work-with-terraform/overview"
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
        <button type="button" class="btn-secondary" @click.prevent="addUser">
          {{ $t("settings.members.add-more") }}
        </button>
      </span>

      <button
        type="button"
        class="btn-primary"
        :disabled="!hasValidUserOnly()"
        @click.prevent="addOrInvite"
      >
        <heroicons-solid:plus v-if="canManageMember" class="h-5 w-5" />
        <heroicons-outline:mail v-else class="mr-2 h-5 w-5" />
        {{ $t(`settings.members.${canManageMember ? "add" : "invites"}`) }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { NSwitch } from "naive-ui";
import RoleSelect from "./RoleSelect.vue";
import {
  Principal,
  PrincipalCreate,
  MemberCreate,
  RoleType,
  UNKNOWN_ID,
} from "../types";
import { isValidEmail, hasWorkspacePermission } from "../utils";
import {
  useUIStateStore,
  featureToRef,
  useCurrentUser,
  usePrincipalStore,
  useMemberStore,
} from "@/store";

type User = {
  email: string;
  role: RoleType;
  isServiceAccount: boolean;
};

interface LocalState {
  userList: User[];
  errorList: string[];
}

export default defineComponent({
  name: "MemberAddOrInvite",
  components: { RoleSelect, NSwitch },
  setup() {
    const memberStore = useMemberStore();

    const currentUser = useCurrentUser();

    const canManageMember = computed(() => {
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-member",
        currentUser.value.role
      );
    });

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const state = reactive<LocalState>({
      userList: [],
      errorList: [],
    });

    for (let i = 0; i < 3; i++) {
      state.userList.push({
        email: "",
        role: "DEVELOPER",
        isServiceAccount: false,
      });
      state.errorList.push("");
    }

    const validateUserInternal = (user: User): string => {
      if (!user.email) {
        return "";
      }
      if (user.isServiceAccount) {
        if (isValidEmail(user.email)) {
          return "Please use name instead of email for service account";
        }
      } else {
        if (!isValidEmail(user.email)) {
          return "Invalid email address";
        } else {
          const member = memberStore.memberByEmail(user.email);
          if (member.id != UNKNOWN_ID) {
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
      state.userList.push({
        email: "",
        role: "DEVELOPER",
        isServiceAccount: false,
      });
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

    const addOrInvite = () => {
      for (const user of state.userList) {
        if (isValidEmail(user.email)) {
          // Needs to assign to a local variable as user will change after createPrincipal but before createdMember
          let role = user.role;
          // If admin feature is NOT enabled, then we set every user to OWNER role.
          if (!hasRBACFeature.value) {
            role = "OWNER";
          }
          // Note "principal/createPrincipal" would return the existing principal.
          // This could happen if another client has just created the principal
          // with this email.
          const newPrincipal: PrincipalCreate = {
            name: user.email.split("@")[0],
            email: user.email,
            type: user.isServiceAccount ? "SERVICE_ACCOUNT" : "END_USER",
          };
          usePrincipalStore()
            .createPrincipal(newPrincipal)
            .then((principal: Principal) => {
              const newMember: MemberCreate = {
                principalId: principal.id,
                status: canManageMember.value ? "ACTIVE" : "INVITED",
                role,
              };
              // Note "principal/createdMember" would return the existing role mapping.
              // This could happen if another client has just created the role mapping with
              // this principal.
              memberStore.createdMember(newMember);
            });

          useUIStateStore().saveIntroStateByKey({
            key: "member.addOrInvite",
            newState: true,
          });
        }
      }
      state.userList.forEach((item) => {
        item.email = "";
        item.role = "DEVELOPER";
        item.isServiceAccount = false;
      });
      state.errorList = [""];
    };

    return {
      state,
      canManageMember,
      hasRBACFeature,
      validateUser,
      clearValidationError,
      addUser,
      hasValidUserOnly,
      addOrInvite,
    };
  },
});
</script>
