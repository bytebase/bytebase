<template>
  <div v-bind="$attrs">
    <NFormItem :label="$t('settings.profile.password')">
      <div class="w-full space-y-1">
        <span
          :class="[
            'flex items-center gap-x-1 textinfolabel text-sm',
            passwordHint ? '!text-error' : '',
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
            url="/setting/general#account"
          />
        </span>
        <NInput
          :value="password"
          type="password"
          :status="passwordHint ? 'error' : undefined"
          :input-props="{ autocomplete: 'new-password' }"
          :placeholder="$t('common.sensitive-placeholder')"
          @update:value="$emit('update:password', $event)"
        />
      </div>
    </NFormItem>
    <NFormItem :label="$t('settings.profile.password-confirm')">
      <div class="w-full flex flex-col justify-start items-start">
        <NInput
          :value="passwordConfirm"
          type="password"
          :status="passwordMismatch ? 'error' : undefined"
          :input-props="{ autocomplete: 'new-password' }"
          :placeholder="$t('settings.profile.password-confirm-placeholder')"
          @update:value="$emit('update:passwordConfirm', $event)"
        />
        <span v-if="passwordMismatch" class="text-error text-sm mt-1 pl-1">
          {{ $t("settings.profile.password-mismatch") }}
        </span>
      </div>
    </NFormItem>
  </div>
</template>

<script lang="tsx" setup>
import { CircleHelpIcon } from "lucide-vue-next";
import { NFormItem, NInput, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { useSettingV1Store } from "@/store";
import { PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";

const props = withDefaults(
  defineProps<{
    password: string;
    passwordConfirm: string;
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

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const passwordRestrictionSetting = computed(
  () =>
    settingV1Store.getSettingByName("bb.workspace.password-restriction")?.value
      ?.passwordRestrictionSetting ?? PasswordRestrictionSetting.fromPartial({})
);

const passwordRestrictionText = computed(() => {
  const text = [
    t("settings.general.workspace.password-restriction.min-length", {
      min: passwordRestrictionSetting.value.minLength,
    }),
  ];
  if (passwordRestrictionSetting.value.requireNumber) {
    text.push(
      t("settings.general.workspace.password-restriction.require-number")
    );
  }
  if (passwordRestrictionSetting.value.requireUppercaseLetter) {
    text.push(
      t(
        "settings.general.workspace.password-restriction.require-uppercase-letter"
      )
    );
  } else if (passwordRestrictionSetting.value.requireLetter) {
    text.push(
      t("settings.general.workspace.password-restriction.require-letter")
    );
  }
  if (passwordRestrictionSetting.value.requireSpecialCharacter) {
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
  if (pwd.length < passwordRestrictionSetting.value.minLength) {
    return true;
  }
  if (passwordRestrictionSetting.value.requireNumber && !/[0-9]+/.test(pwd)) {
    return true;
  }
  if (
    passwordRestrictionSetting.value.requireLetter &&
    !/[a-zA-Z]+/.test(pwd)
  ) {
    return true;
  }
  if (
    passwordRestrictionSetting.value.requireUppercaseLetter &&
    !/[A-Z]+/.test(pwd)
  ) {
    return true;
  }
  if (
    passwordRestrictionSetting.value.requireSpecialCharacter &&
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
