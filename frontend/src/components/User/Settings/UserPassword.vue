<template>
  <div class="flex flex-col gap-y-6" v-bind="$attrs">
    <div :label="$t('settings.profile.password')">
      <div>
        <label class="block text-sm font-medium leading-5 text-control">
          {{ $t("settings.profile.password") }}
          <RequiredStar />
        </label>
        <span
          :class="[
            'flex items-center gap-x-1 textinfolabel text-sm!',
            passwordHint ? 'text-error!' : '',
          ]"
        >
          {{ $t("settings.profile.password-hint") }}
          <NTooltip>
            <template #trigger>
              <CircleHelpIcon class="w-4" />
            </template>
            <component :is="passwordRestrictionText" class="text-sm" />
          </NTooltip>
          <LearnMoreLink
            v-if="showLearnMore"
            :external="false"
            class="text-sm!"
            :url="
              router.resolve({
                name: SETTING_ROUTE_WORKSPACE_GENERAL,
                hash: '#account',
              }).fullPath
            "
          />
        </span>
      </div>
      <div class="w-full flex flex-col gap-y-1">
        <div class="mt-1 relative flex flex-row items-center">
          <NInput
            :value="password"
            :type="showPassword ? 'text' : 'password'"
            :status="passwordHint ? 'error' : undefined"
            :input-props="{ autocomplete: 'new-password' }"
            :placeholder="$t('common.sensitive-placeholder')"
            @update:value="$emit('update:password', $event)"
          />
          <div
            class="hover:cursor-pointer absolute right-3"
            @click="
              () => {
                showPassword = !showPassword;
              }
            "
          >
            <EyeIcon v-if="showPassword" class="w-4 h-4" />
            <EyeOffIcon v-else class="w-4 h-4" />
          </div>
        </div>
      </div>
    </div>
    <div :label="$t('settings.profile.password-confirm')">
      <label
        for="email"
        class="block text-sm font-medium leading-5 text-control"
      >
        {{ $t("settings.profile.password-confirm") }}
        <RequiredStar />
      </label>
      <div class="w-full mt-1 flex flex-col justify-start items-start">
        <div class="w-full relative flex flex-row items-center">
          <NInput
            :value="passwordConfirm"
            :type="showPassword ? 'text' : 'password'"
            :status="passwordMismatch ? 'error' : undefined"
            :input-props="{ autocomplete: 'new-password' }"
            :placeholder="$t('settings.profile.password-confirm-placeholder')"
            @update:value="$emit('update:passwordConfirm', $event)"
          />
          <div
            class="hover:cursor-pointer absolute right-3"
            @click="
              () => {
                showPassword = !showPassword;
              }
            "
          >
            <EyeIcon v-if="showPassword" class="w-4 h-4" />
            <EyeOffIcon v-else class="w-4 h-4" />
          </div>
        </div>
        <span v-if="passwordMismatch" class="text-error text-sm mt-1 pl-1">
          {{ $t("settings.profile.password-mismatch") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import {
  CircleAlertIcon,
  CircleCheckIcon,
  CircleHelpIcon,
  EyeIcon,
  EyeOffIcon,
} from "lucide-vue-next";
import { NInput, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { type WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";

const props = withDefaults(
  defineProps<{
    password: string;
    passwordConfirm: string;
    passwordRestriction: WorkspaceProfileSetting_PasswordRestriction;
    showLearnMore?: boolean;
  }>(),
  {
    showLearnMore: true,
  }
);

defineEmits<{
  (event: "update:password", password: string): void;
  (event: "update:passwordConfirm", passwordConfirm: string): void;
}>();

const showPassword = ref<boolean>(false);
const { t } = useI18n();
const router = useRouter();

const passwordCheck = computed(() => {
  const check: { text: string; matched: boolean }[] = [
    {
      text: t("settings.general.workspace.password-restriction.min-length", {
        min: props.passwordRestriction.minLength,
      }),
      matched: props.password.length >= props.passwordRestriction.minLength,
    },
  ];

  if (props.passwordRestriction.requireNumber) {
    check.push({
      text: t("settings.general.workspace.password-restriction.require-number"),
      matched: /[0-9]+/.test(props.password),
    });
  }
  if (props.passwordRestriction.requireUppercaseLetter) {
    check.push({
      text: t(
        "settings.general.workspace.password-restriction.require-uppercase-letter"
      ),
      matched: /[A-Z]+/.test(props.password),
    });
  } else if (props.passwordRestriction.requireLetter) {
    check.push({
      text: t("settings.general.workspace.password-restriction.require-letter"),
      matched: /[a-zA-Z]+/.test(props.password),
    });
  }
  if (props.passwordRestriction.requireSpecialCharacter) {
    check.push({
      text: t(
        "settings.general.workspace.password-restriction.require-special-character"
      ),
      matched: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]+/.test(props.password),
    });
  }

  return check;
});

const passwordRestrictionText = computed(() => {
  return (
    <ul class="list-disc">
      {passwordCheck.value.map((check, i) => (
        <li key={i} class={"flex gap-x-1 items-center"}>
          {check.matched ? (
            <CircleCheckIcon class={"w-4 text-green-400"} />
          ) : (
            <CircleAlertIcon class={"w-4 text-red-400"} />
          )}
          {check.text}
        </li>
      ))}
    </ul>
  );
});

const passwordHint = computed(() => {
  const pwd = props.password;
  if (!pwd) {
    return false;
  }
  return passwordCheck.value.some((check) => !check.matched);
});

const passwordMismatch = computed(() => {
  if (!props.password) {
    return false;
  }
  return props.password !== props.passwordConfirm;
});

defineExpose({
  passwordHint,
  passwordMismatch,
});
</script>
