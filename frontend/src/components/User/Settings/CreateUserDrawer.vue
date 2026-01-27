<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="
        isEditMode
          ? $t('settings.members.update-user')
          : $t('settings.members.add-user')
      "
    >
      <template #default>
        <div class="flex flex-col gap-y-6">
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.name") }}
            </label>
            <NInput
              v-model:value="state.user.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="Foo"
              :maxlength="200"
              :disabled="!allowUpdate"
            />
          </div>

          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.email") }}
              <RequiredStar class="ml-0.5" />
            </label>
            <EmailInput
              v-model:value="state.user.email"
              :domain="''"
              :show-domain="false"
              :disabled="isEditMode"
            />
          </div>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="['bb.workspaces.setIamPolicy']"
          >
            <div class="flex flex-col gap-y-2">
              <div>
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.members.table.roles") }}
                </label>
              </div>
              <RoleSelect
                v-model:value="state.roles"
                :multiple="true"
                :disabled="slotProps.disabled"
              />
            </div>
          </PermissionGuardWrapper>

          <div class="flex flex-col gap-y-2">
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
              :disabled="!allowUpdate"
            />
          </div>

          <UserPassword
            ref="userPasswordRef"
            v-model:password="state.user.password"
            v-model:password-confirm="state.passwordConfirm"
            :password-restriction="passwordRestrictionSetting"
            :disabled="!allowUpdate"
          />
        </div>
      </template>
      <template #footer>
        <div class="w-full flex flex-row items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="[isEditMode ? 'bb.users.update' : 'bb.users.create']"
          >
            <NButton
              type="primary"
              :disabled="!allowConfirm || slotProps.disabled"
              :loading="state.isRequesting"
              @click="createOrUpdateUser"
            >
              {{ isEditMode ? $t("common.update") : $t("common.confirm") }}
            </NButton>
          </PermissionGuardWrapper>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  getUserFullNameByType,
  pushNotification,
  useSettingV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import { getUserEmailInBinding, UNKNOWN_USER_NAME, unknownUser } from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import UserPassword from "./UserPassword.vue";

interface LocalState {
  isRequesting: boolean;
  user: User;
  roles: string[];
  passwordConfirm: string;
}

const props = defineProps<{
  user?: User;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
  (event: "updated", user: User): void;
}>();

const workspaceStore = useWorkspaceV1Store();

const { t } = useI18n();
const userStore = useUserStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const state = reactive<LocalState>({
  isRequesting: false,
  user: unknownUser(),
  roles: [PresetRoleType.WORKSPACE_MEMBER],
  passwordConfirm: "",
});

const isEditMode = computed(
  () => !!props.user && props.user.name !== unknownUser().name
);

const allowUpdate = computed(() => {
  if (!isEditMode.value) {
    return true;
  }
  return hasWorkspacePermissionV2("bb.users.update");
});

const initialRoles = computed(() => {
  if (!props.user || props.user.name === UNKNOWN_USER_NAME) {
    return [PresetRoleType.WORKSPACE_MEMBER];
  }

  const roles = workspaceStore.userMapToRoles.get(
    getUserFullNameByType(props.user)
  );
  return roles ? [...roles] : [];
});

watch(
  () => props.user,
  (user) => {
    if (!user) {
      return;
    }
    state.user = cloneDeep(create(UserSchema, user));
    state.roles = [...initialRoles.value];
  },
  {
    immediate: true,
  }
);

const passwordRestrictionSetting = computed(
  () => useSettingV1Store().workspaceProfile.passwordRestriction
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
  return email;
};

const createOrUpdateUser = async () => {
  state.isRequesting = true;
  try {
    if (isEditMode.value) {
      await updateUser();
    } else {
      await createUser();
    }
  } catch {
    // nothing
  } finally {
    state.isRequesting = false;
  }
};

const createUser = async () => {
  const createdUser = await userStore.createUser({
    ...state.user,
    userType: UserType.USER,
    title: state.user.title || extractUserTitle(state.user.email),
    password: state.user.password,
  });

  if (state.roles.length > 0) {
    await workspaceStore.patchIamPolicy([
      {
        member: getUserEmailInBinding(createdUser.email),
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

const updateUser = async () => {
  const user = props.user;
  if (!user) {
    return;
  }

  const updateMask: string[] = [];
  const payload = create(UserSchema, state.user);
  if (payload.title !== user.title) {
    updateMask.push("title");
  }
  if (payload.phone !== user.phone) {
    updateMask.push("phone");
  }
  if (payload.password) {
    updateMask.push("password");
  }

  let updatedUser: User = user;

  if (updateMask.length > 0) {
    updatedUser = await userStore.updateUser(
      create(UpdateUserRequestSchema, {
        user: payload,
        updateMask: create(FieldMaskSchema, {
          paths: updateMask,
        }),
      })
    );
  }

  if (!isEqual([...initialRoles.value].sort(), [...state.roles].sort())) {
    await workspaceStore.patchIamPolicy([
      {
        member: getUserEmailInBinding(updatedUser.email),
        roles: state.roles,
      },
    ]);
  }

  emit("updated", updatedUser);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  emit("close");
};
</script>
