<template>
  <div class="mb-7 mt-4 lg:mt-0">
    <p
      class="font-medium flex flex-row justify-start items-center mb-2 gap-x-2"
    >
      {{ $t("settings.general.workspace.password-restriction.self") }}
      <FeatureBadge :feature="PlanFeature.FEATURE_PASSWORD_RESTRICTIONS" />
    </p>
    <div class="w-full flex flex-col gap-y-3">
      <div class="flex items-center">
        <NInputNumber
          :value="state.minLength"
          :disabled="!allowEdit"
          class="w-24 mr-2"
          :min="1"
          :placeholder="'Minimum length'"
          :precision="0"
          size="small"
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
        {{
          $t("settings.general.workspace.password-restriction.min-length", {
            min: state.minLength || DEFAULT_MIN_LENGTH,
          })
        }}
      </div>
      <NCheckbox
        :checked="state.requireNumber"
        :disabled="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireNumber: val });
          }
        "
      >
        {{
          $t("settings.general.workspace.password-restriction.require-number")
        }}
      </NCheckbox>
      <NCheckbox
        :checked="state.requireLetter"
        :disabled="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireLetter: val });
          }
        "
      >
        {{
          $t("settings.general.workspace.password-restriction.require-letter")
        }}
      </NCheckbox>
      <NCheckbox
        :checked="state.requireUppercaseLetter"
        :disabled="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireUppercaseLetter: val });
          }
        "
      >
        {{
          $t(
            "settings.general.workspace.password-restriction.require-uppercase-letter"
          )
        }}
      </NCheckbox>
      <NCheckbox
        :checked="state.requireSpecialCharacter"
        :disabled="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireSpecialCharacter: val });
          }
        "
      >
        {{
          $t(
            "settings.general.workspace.password-restriction.require-special-character"
          )
        }}
      </NCheckbox>
      <NCheckbox
        :checked="state.requireResetPasswordForFirstLogin"
        :disabled="!allowEdit"
        @update:checked="
          (val) => {
            onUpdate({ requireResetPasswordForFirstLogin: val });
          }
        "
      >
        {{
          $t(
            "settings.general.workspace.password-restriction.require-reset-password-for-first-login"
          )
        }}
      </NCheckbox>
      <NCheckbox
        :checked="!!state.passwordRotation"
        :disabled="!allowEdit"
        class="flex! items-center!"
        @update:checked="
          (checked) => {
            if (checked) {
              onUpdate({
                passwordRotation: create(DurationSchema, {
                  seconds: BigInt(7 * 24 * 60 * 60) /* default 7 days */,
                  nanos: 0,
                }),
              });
            } else {
              delete state.passwordRotation;
            }
          }
        "
      >
        <i18n-t
          tag="div"
          keypath="settings.general.workspace.password-restriction.password-rotation"
          class="flex items-center gap-x-2"
        >
          <template #day>
            <NInputNumber
              v-if="state.passwordRotation"
              :value="Number(state.passwordRotation.seconds) / (24 * 60 * 60)"
              :disabled="!allowEdit"
              :min="1"
              class="w-24 mx-2"
              :size="'small'"
              :placeholder="'Minimum length'"
              :precision="0"
              @click="(e: MouseEvent) => e.stopPropagation()"
              @update:value="
                (val) =>
                  onUpdate({
                    passwordRotation: create(DurationSchema, {
                      seconds: BigInt((val || 1) * 24 * 60 * 60),
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
    :feature="PlanFeature.FEATURE_PASSWORD_RESTRICTIONS"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { NCheckbox, NInputNumber } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceProfileSetting_PasswordRestrictionSchema } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";

const DEFAULT_MIN_LENGTH = 8;

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const showFeatureModal = ref<boolean>(false);
const hasPasswordFeature = featureToRef(
  PlanFeature.FEATURE_PASSWORD_RESTRICTIONS
);

const passwordRestrictionSetting = computed(() => {
  return (
    settingV1Store.workspaceProfileSetting?.passwordRestriction ??
    create(WorkspaceProfileSetting_PasswordRestrictionSchema, {})
  );
});

const state = reactive<WorkspaceProfileSetting_PasswordRestriction>(
  cloneDeep(passwordRestrictionSetting.value)
);

const onUpdate = async (
  update: Partial<WorkspaceProfileSetting_PasswordRestriction>
) => {
  if (!hasPasswordFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  Object.assign(state, update);
};

defineExpose({
  isDirty: computed(() => !isEqual(passwordRestrictionSetting.value, state)),
  update: async () => {
    if (!settingV1Store.workspaceProfileSetting) {
      return;
    }
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        passwordRestriction: create(
          WorkspaceProfileSetting_PasswordRestrictionSchema,
          {
            ...state,
          }
        ),
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.password_restriction"],
      }),
    });
  },
  revert: () => {
    Object.assign(state, passwordRestrictionSetting.value);
  },
});
</script>
