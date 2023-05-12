import { computed, Ref, unref, watch } from "vue";
import { DatabaseLabel, MaybeRef } from "@/types";
import {
  getLabelValueFromLabelList,
  parseDatabaseLabelValueByTemplate,
  setLabelValue,
} from "@/utils";
import { type Project, TenantMode } from "@/types/proto/v1/project_service";
export const useDBNameTemplateInputState = (
  project: MaybeRef<Project>,
  values: {
    databaseName: Ref<string>;
    labels: Ref<DatabaseLabel[]>;
  }
) => {
  const TENANT_LABEL_KEY = "bb.tenant";
  const TENANT_LABEL_REGEXP_GROUP_NAME = "TENANT";

  const tenantValue = computed(() => {
    return getLabelValueFromLabelList(unref(values.labels), TENANT_LABEL_KEY);
  });

  const parsedTenantValue = computed(() => {
    return parseDatabaseLabelValueByTemplate(
      unref(project).dbNameTemplate,
      values.databaseName.value,
      TENANT_LABEL_REGEXP_GROUP_NAME
    );
  });

  watch(
    parsedTenantValue,
    (newValue, oldValue) => {
      const proj = unref(project);
      if (proj.tenantMode !== TenantMode.TENANT_MODE_ENABLED) return;
      if (!proj.dbNameTemplate) return;

      const tenant = tenantValue.value;
      if (tenant && tenant !== oldValue) {
        // Tenant value has been changed by user manually, don't touch it.
        return;
      }
      setLabelValue(values.labels.value, TENANT_LABEL_KEY, newValue);
    },
    { immediate: true }
  );
};
