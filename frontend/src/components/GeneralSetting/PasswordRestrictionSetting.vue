<template>
  <div class="mb-7 mt-4 lg:mt-0">
    <p class="font-medium flex flex-row justify-start items-center mb-2">
      <span class="mr-2">
        {{ $t("settings.general.workspace.password-restriction.self") }}
      </span>
      <FeatureBadge feature="bb.feature.password-restriction" />
    </p>
    <div class="w-full flex flex-col space-y-3">
      <div class="flex items-center space-x-2">
        <NInputNumber
          :value="passwordRestrictionSetting.minLength"
          :readonly="!allowEdit"
          class="w-24"
          :min="1"
          :placeholder="'Minimum length'"
          :precision="0"
          @update:value="
            (val) =>
              onUpdate({
                minLength: Math.max(
                  val || DEFAULT_MIN_LENGTH,
                  DEFAULT_MIN_LENGTH
                ),
              })
          "
        />
        <span class="textlabel">
          {{
            $t("settings.general.workspace.password-restriction.min-length", {
              min: passwordRestrictionSetting.minLength || DEFAULT_MIN_LENGTH,
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
      <NCheckbox
        :checked="passwordRestrictionSetting.requireResetPasswordForFirstLogin"
        :readonly="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireResetPasswordForFirstLogin: val });
          }
        "
      >
        <span class="textlabel">
          {{
            $t(
              "settings.general.workspace.password-restriction.require-reset-password-for-first-login"
            )
          }}
        </span>
      </NCheckbox>
      <NCheckbox
        :checked="!!passwordRestrictionSetting.passwordRotation"
        :readonly="!allowEdit"
        class="!flex !items-center"
        @update:checked="
          (checked) => {
            onUpdate({
              passwordRotation: checked
                ? Duration.fromPartial({
                    seconds: 7 * 24 * 60 * 60 /* default 7 days */,
                    nanos: 0,
                  })
                : undefined,
            });
          }
        "
      >
        <i18n-t
          tag="div"
          keypath="settings.general.workspace.password-restriction.password-rotation"
          class="flex items-center space-x-2 textlabel"
        >
          <template #day>
            <NInputNumber
              v-if="passwordRestrictionSetting.passwordRotation"
              :value="
                Number(
                  passwordRestrictionSetting.passwordRotation.seconds.divide(
                    24 * 60 * 60
                  )
                )
              "
              :readonly="!allowEdit"
              :min="1"
              class="w-24 mx-2"
              :size="'small'"
              :placeholder="'Minimum length'"
              :precision="0"
              @click="(e: MouseEvent) => e.stopPropagation()"
              @update:value="
                (val) =>
                  onUpdate({
                    passwordRotation: Duration.fromPartial({
                      seconds: (val || 1) * 24 * 60 * 60,
                      nanos: 0,
                    }),
                  })
              "
            />
            <span v-else class="mx-1">N</span>
          </template>
        </i18n-t>
      </NCheckbox>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.password-restriction"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="tsx">
import { NInputNumber, NCheckbox } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { featureToRef, pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";

const DEFAULT_MIN_LENGTH = 8;

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const showFeatureModal = ref<boolean>(false);
const hasPasswordFeature = featureToRef("bb.feature.password-restriction");

const passwordRestrictionSetting = computed(
  () =>
    settingV1Store.getSettingByName("bb.workspace.password-restriction")?.value
      ?.passwordRestrictionSetting ?? PasswordRestrictionSetting.fromPartial({})
);

const onUpdate = async (update: Partial<PasswordRestrictionSetting>) => {
  if (!hasPasswordFeature.value) {
    showFeatureModal.value = true;
    return;
  }
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
