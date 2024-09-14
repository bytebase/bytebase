<template>
  <div class="space-y-6" v-bind="$attrs">
    <div :label="$t('settings.profile.password')">
      <div class="flex items-center space-x-2">
        <label class="block text-sm font-medium leading-5 text-control">
          {{ $t("settings.profile.password") }}
        </label>
        <span
          :class="[
            'flex items-center gap-x-1 textinfolabel !text-sm',
            passwordHint ? '!text-error' : '',
          ]"
        >
          {{ $t("settings.profile.password-hint") }}
          <NTooltip>
            <template #trigger>
              <CircleHelpIcon class="w-3" />
            </template>
            <component :is="passwordRestrictionText" class="text-sm" />
          </NTooltip>
          <LearnMoreLink
            v-if="showLearnMore"
            :external="false"
            class="!text-sm"
            url="/setting/general#account"
          />
        </span>
      </div>
      <div class="w-full space-y-1">
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
import { CircleHelpIcon, EyeIcon, EyeOffIcon } from "lucide-vue-next";
import { NInput, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { type PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";

const props = withDefaults(
  defineProps<{
    password: string;
    passwordConfirm: string;
    passwordRestriction: PasswordRestrictionSetting;
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

const passwordRestrictionText = computed(() => {
  const text = [
    t("settings.general.workspace.password-restriction.min-length", {
      min: props.passwordRestriction.minLength,
    }),
  ];
  if (props.passwordRestriction.requireNumber) {
    text.push(
      t("settings.general.workspace.password-restriction.require-number")
    );
  }
  if (props.passwordRestriction.requireUppercaseLetter) {
    text.push(
      t(
        "settings.general.workspace.password-restriction.require-uppercase-letter"
      )
    );
  } else if (props.passwordRestriction.requireLetter) {
    text.push(
      t("settings.general.workspace.password-restriction.require-letter")
    );
  }
  if (props.passwordRestriction.requireSpecialCharacter) {
    text.push(
      t(
        "settings.general.workspace.password-restriction.require-special-character"
      )
    );
  }

  return (
    <ul class="list-disc px-2">
      {text.map((t, i) => (
        <li key={i}>{t}</li>
      ))}
    </ul>
  );
});

const passwordHint = computed(() => {
  const pwd = props.password;
  if (!pwd) {
    return false;
  }
  if (pwd.length < props.passwordRestriction.minLength) {
    return true;
  }
  if (props.passwordRestriction.requireNumber && !/[0-9]+/.test(pwd)) {
    return true;
  }
  if (props.passwordRestriction.requireLetter && !/[a-zA-Z]+/.test(pwd)) {
    return true;
  }
  if (props.passwordRestriction.requireUppercaseLetter && !/[A-Z]+/.test(pwd)) {
    return true;
  }
  if (
    props.passwordRestriction.requireSpecialCharacter &&
    // eslint-disable-next-line no-useless-escape
    !/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+/.test(pwd)
  ) {
    return true;
  }
  return false;
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
