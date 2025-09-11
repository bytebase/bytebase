<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="$t('settings.members.add-user')"
    >
      <template #default>
        <div class="space-y-6">
          <div class="w-full mb-4 space-y-2">
            <div class="flex items-center space-x-1">
              <div class="text-sm font-medium">{{ $t("common.type") }}</div>
              <a
                href="https://docs.bytebase.com/get-started/terraform?source=console"
                target="_blank"
              >
                <heroicons-outline:question-mark-circle class="w-4 h-4" />
              </a>
            </div>
            <NRadioGroup v-model:value="state.user.userType">
              <NRadio :value="UserType.USER" :label="$t('common.user')" />
              <NRadio
                :value="UserType.SERVICE_ACCOUNT"
                :label="$t('settings.members.service-account')"
              />
            </NRadioGroup>
          </div>

          <div class="space-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.name") }}
            </label>
            <NInput
              v-model:value="state.user.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="Foo"
            />
          </div>

          <div class="space-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.email") }}
              <RequiredStar class="ml-0.5" />
            </label>
            <EmailInput
              v-model:value="state.user.email"
              :domain-prefix="
                state.user.userType === UserType.SERVICE_ACCOUNT
                  ? 'service'
                  : ''
              "
              :fallback-domain="
                state.user.userType === UserType.SERVICE_ACCOUNT
                  ? 'bytebase.com'
                  : ''
              "
              :show-domain="state.user.userType === UserType.SERVICE_ACCOUNT"
            />
          </div>

          <div class="space-y-2">
            <div>
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("settings.members.table.roles") }}
              </label>
            </div>
            <RoleSelect v-model:value="state.roles" :multiple="true" />
          </div>

          <template v-if="state.user.userType === UserType.USER">
            <div class="space-y-2">
              <div>
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.profile.phone") }}
                </label>
                <span class="textinfolabel text-sm">
                  {{ $t("settings.profile.phone-tips") }}
                </span>
              </div>
              <NInput
                v-model:value="state.user.phone"
                type="text"
                :input-props="{
                  type: 'tel',
                  autocomplete: 'new-password',
                }"
              />
            </div>

            <UserPassword
              ref="userPasswordRef"
              v-model:password="state.user.password"
              v-model:password-confirm="state.passwordConfirm"
              :password-restriction="passwordRestrictionSetting"
            />
          </template>
        </div>
      </template>
      <template #footer>
        <div class="w-full flex flex-row items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowConfirm"
            :loading="state.isRequesting"
            @click="tryCreateOrUpdateUser"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NInput, NRadioGroup, NRadio } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  pushNotification,
  useSettingV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import { emptyUser, PresetRoleType } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import UserPassword from "./UserPassword.vue";

interface LocalState {
  isRequesting: boolean;
  user: User;
  roles: string[];
  passwordConfirm: string;
}

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
}>();

const workspaceStore = useWorkspaceV1Store();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const userStore = useUserStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const state = reactive<LocalState>({
  isRequesting: false,
  user: emptyUser(),
  roles: [PresetRoleType.WORKSPACE_MEMBER],
  passwordConfirm: "",
});

const passwordRestrictionSetting = computed(
  () => settingV1Store.passwordRestriction
);

const allowConfirm = computed(() => {
  if (!state.user.email) {
    return false;
  }
  if (userPasswordRef.value?.passwordHint) {
    return false;
  }
  if (userPasswordRef.value?.passwordMismatch) {
    return false;
  }

  return true;
});

const extractUserTitle = (email: string): string => {
  const atIndex = email.indexOf("@");
  if (atIndex !== -1) {
    return email.substring(0, atIndex);
  }
  // If there is no @, we just return the email as title.
  return email;
};

const tryCreateOrUpdateUser = async () => {
  const createdUser = await userStore.createUser({
    ...state.user,
    title: state.user.title || extractUserTitle(state.user.email),
    password: state.user.password,
  });
  if (state.roles.length > 0) {
    await workspaceStore.patchIamPolicy([
      {
        member: `user:${createdUser.email}`,
        roles: state.roles,
      },
    ]);
  }
  emit("created", createdUser);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.created"),
  });
  emit("close");
};
</script>
