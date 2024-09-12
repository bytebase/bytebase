<template>
  <div class="mb-7 mt-4 lg:mt-0">
    <p class="font-medium flex flex-row justify-start items-center mb-2">
      <span class="mr-2">
        {{ $t("settings.general.workspace.password-restriction.self") }}
      </span>
    </p>
    <div class="w-full flex flex-col space-y-3">
      <div class="flex items-center space-x-2">
        <NInputNumber
          :value="passwordRestrictionSetting.minLength"
          :readonly="!allowEdit"
          class="w-24"
          :min="DEFAULT_MIN_LENGTH"
          :placeholder="'Minimum length'"
          :precision="0"
          @update:value="
            (val) => onUpdate({ minLength: val || DEFAULT_MIN_LENGTH })
          "
        />
        <span class="textlabel">
          {{
            $t("settings.general.workspace.password-restriction.min-length", {
              min: DEFAULT_MIN_LENGTH,
            })
          }}
        </span>
      </div>
      <NCheckbox
        :checked="passwordRestrictionSetting.requireNumber"
        :readonly="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireNumber: val });
          }
        "
      >
        <span class="textlabel">
          {{
            $t("settings.general.workspace.password-restriction.require-number")
          }}
        </span>
      </NCheckbox>
      <NCheckbox
        :checked="passwordRestrictionSetting.requireLetter"
        :readonly="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireLetter: val });
          }
        "
      >
        <span class="textlabel">
          {{
            $t("settings.general.workspace.password-restriction.require-letter")
          }}
        </span>
      </NCheckbox>
      <NCheckbox
        :checked="passwordRestrictionSetting.requireUppercaseLetter"
        :readonly="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireUppercaseLetter: val });
          }
        "
      >
        <span class="textlabel">
          {{
            $t(
              "settings.general.workspace.password-restriction.require-uppercase-letter"
            )
          }}
        </span>
      </NCheckbox>
      <NCheckbox
        :checked="passwordRestrictionSetting.requireSpecialCharacter"
        :readonly="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireSpecialCharacter: val });
          }
        "
      >
        <span class="textlabel">
          {{
            $t(
              "settings.general.workspace.password-restriction.require-special-character"
            )
          }}
        </span>
      </NCheckbox>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { NInputNumber, NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";

const DEFAULT_MIN_LENGTH = 8;

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const passwordRestrictionSetting = computed(
  () =>
    settingV1Store.getSettingByName("bb.workspace.password-restriction")?.value
      ?.passwordRestrictionSetting ?? PasswordRestrictionSetting.fromPartial({})
);

const onUpdate = async (update: Partial<PasswordRestrictionSetting>) => {
  await settingV1Store.upsertSetting({
    name: "bb.workspace.password-restriction",
    value: {
      passwordRestrictionSetting: {
        ...passwordRestrictionSetting.value,
        ...update,
      },
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
