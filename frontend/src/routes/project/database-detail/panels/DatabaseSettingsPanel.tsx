import { useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
import {
  LabelListEditor,
  type LabelListEditorItem,
} from "@/components/LabelListEditor";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  convertLabelsToKVList,
  getDatabaseProject,
  getInstanceResource,
  hasProjectPermissionV2,
} from "@/utils";
import {
  buildDatabaseEnvironmentUpdate,
  buildDatabaseLabelsUpdate,
  labelMap,
  labelsChanged,
  labelsFingerprint,
  validRows,
} from "./databaseSettingsActions";
import { useDatabaseSettingsDom } from "./useDatabaseSettingsDom";

export function DatabaseSettingsPanel({ database }: { database: Database }) {
  const actor = useAppStore((s) => s.currentUser?.name ?? "");
  const workspace = useAppStore((s) => s.workspaceResourceName());
  return (
    <SettingsForm
      key={JSON.stringify([actor, workspace, database.name, database.project])}
      database={database}
      actor={actor}
      workspace={workspace}
    />
  );
}

function SettingsForm({
  database,
  actor,
  workspace,
}: {
  database: Database;
  actor: string;
  workspace: string;
}) {
  const { t } = useTranslation();
  const labels = database.labels;
  const instanceEnvironment = getInstanceResource(database).environment ?? "";
  const allowClearEnvironment = !instanceEnvironment;
  const [savedLabels, setSavedLabels] = useState(labels);
  const [kvList, setKVList] = useState<LabelListEditorItem[]>(() =>
    convertLabelsToKVList(labels)
  );
  const [isUpdatingLabels, setIsUpdatingLabels] = useState(false);
  if (
    savedLabels !== labels &&
    labelsFingerprint(savedLabels) !== labelsFingerprint(labels)
  ) {
    setSavedLabels(labels);
    if (!labelsChanged(savedLabels, kvList))
      setKVList(convertLabelsToKVList(labels));
  }
  const project = getDatabaseProject(database);
  const allowUpdateDatabase =
    !!project && hasProjectPermissionV2(project, "bb.databases.update");
  const dirty = labelsChanged(labels, kvList);
  const allowSave = dirty && validRows(kvList) && !isUpdatingLabels;
  const dom = useDatabaseSettingsDom({
    database,
    actor,
    workspace,
    rows: kvList,
    setRows: setKVList,
    allowed: allowUpdateDatabase,
  });

  const handleSelectEnvironment = async (name?: string) => {
    const nextEnvironment = name || "";

    if (!nextEnvironment && instanceEnvironment) {
      return;
    }
    if (nextEnvironment === database.effectiveEnvironment) {
      return;
    }

    const finish = dom.startNative("environment");
    try {
      await useAppStore
        .getState()
        .updateDatabase(
          buildDatabaseEnvironmentUpdate(database.name, nextEnvironment)
        );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      finish();
    }
  };

  const handleCancel = () => {
    dom.edit(convertLabelsToKVList(labels));
  };

  const handleSave = async () => {
    if (!allowSave) {
      return;
    }

    setIsUpdatingLabels(true);
    const finish = dom.startNative("labels");
    try {
      await useAppStore
        .getState()
        .updateDatabase(
          buildDatabaseLabelsUpdate(database.name, labelMap(kvList))
        );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      finish();
      setIsUpdatingLabels(false);
    }
  };

  return (
    <div className="divide-y">
      <div className="flex flex-col gap-y-4 pb-7">
        <FormField title={t("common.environment")}>
          <PermissionGuard
            permissions={["bb.databases.update"]}
            project={getDatabaseProject(database)}
          >
            <EnvironmentSelect
              triggerRef={dom.environment}
              className="max-w-md"
              value={database.effectiveEnvironment || ""}
              clearable={allowClearEnvironment}
              disabled={!allowUpdateDatabase}
              renderSuffix={(environment) =>
                instanceEnvironment === environment.name ? (
                  <span className="text-xs text-control-placeholder">
                    ({t("common.default")})
                  </span>
                ) : null
              }
              onChange={handleSelectEnvironment}
            />
          </PermissionGuard>
        </FormField>
      </div>
      <div className="flex flex-col gap-y-4 pt-7">
        <div className="flex items-center">
          <div className="flex-1">
            <div className="flex items-center">
              <p className="flex text-lg font-medium leading-7 text-main">
                {t("database.labels")}
              </p>
            </div>
          </div>
        </div>
        <div className="max-w-120">
          <LabelListEditor
            controlRef={dom.controlRef}
            kvList={kvList}
            onChange={dom.edit}
            readonly={!allowUpdateDatabase}
            showErrors={dirty}
          />
        </div>
        {dirty && (
          <div className="flex items-center justify-end gap-x-2">
            <Button
              ref={dom.revert}
              appearance="outline"
              onClick={handleCancel}
            >
              {t("common.revert")}
            </Button>
            <PermissionGuard
              permissions={["bb.databases.update"]}
              project={getDatabaseProject(database)}
            >
              <Button
                ref={dom.save}
                disabled={!allowSave}
                onClick={() => void handleSave()}
              >
                {t("common.save")}
              </Button>
            </PermissionGuard>
          </div>
        )}
      </div>
    </div>
  );
}
