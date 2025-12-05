<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img class="h-12 w-auto" src="@/assets/logo-full.svg" alt="Bytebase" />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.password-reset.title") }}
      </h2>
      <p class="textinfo mt-2">
        {{ $t("auth.password-reset.content") }}
      </p>
    </div>

    <div class="mt-8">
      <div class="mt-6 flex flex-col gap-y-6">
        <UserPassword
          ref="userPasswordRef"
          v-model:password="state.password"
          v-model:password-confirm="state.passwordConfirm"
          :show-learn-more="false"
          :password-restriction="passwordRestrictionSetting"
        />
        <NButton
          type="primary"
          size="large"
          style="width: 100%"
          :disabled="!allowConfirm"
          @click="onConfirm"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import UserPassword from "@/components/User/Settings/UserPassword.vue";
import {
  pushNotification,
  useAuthStore,
  useCurrentUserV1,
  useSettingV1Store,
  useUserStore,
} from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";

interface LocalState {
  password: string;
  passwordConfirm: string;
}

const state = reactive<LocalState>({
  password: "",
  passwordConfirm: "",
});

const { t } = useI18n();
const me = useCurrentUserV1();
const userStore = useUserStore();
const authStore = useAuthStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();
const router = useRouter();
const settingV1Store = useSettingV1Store();

const passwordRestrictionSetting = computed(
  () => settingV1Store.passwordRestriction
);

const redirectQuery = computed(() => {
  const query = new URLSearchParams(window.location.search);
  return query.get("redirect") || "/";
});

onMounted(async () => {
  if (!authStore.requireResetPassword) {
    router.replace(redirectQuery.value);
    return;
  }
  await settingV1Store.getOrFetchSettingByName(
    Setting_SettingName.WORKSPACE_PROFILE
  );
});

const allowConfirm = computed(() => {
  if (!state.password) {
    return false;
  }
  return (
    !userPasswordRef.value?.passwordHint &&
    !userPasswordRef.value?.passwordMismatch
  );
});

const onConfirm = async () => {
  const patch: User = {
    ...me.value,
    password: state.password,
  };
  await userStore.updateUser(
    createProto(UpdateUserRequestSchema, {
      user: patch,
      updateMask: createProto(FieldMaskSchema, {
        paths: ["password"],
      }),
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  authStore.setRequireResetPassword(false);
  router.replace(redirectQuery.value);
};
</script>
