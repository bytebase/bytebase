import { create } from "@bufbuild/protobuf";
import {
  type ChangeEvent,
  type DragEvent,
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useSubscriptionV1Store,
  useWorkspaceV1Store,
} from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import type { SectionHandle } from "./useSettingSection";

const MAX_FILE_SIZE_MIB = 2;
const SUPPORT_IMAGE_EXTENSIONS = [".jpg", ".jpeg", ".png", ".webp", ".svg"];

interface BrandingSectionProps {
  title: string;
  onDirtyChange: () => void;
}

const convertFileToBase64 = (file: File): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = (error) => reject(error);
  });

export const BrandingSection = forwardRef<SectionHandle, BrandingSectionProps>(
  function BrandingSection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();
    const workspaceStore = useWorkspaceV1Store();

    const workspace = useVueState(() => workspaceStore.currentWorkspace);
    const hasBrandingFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(PlanFeature.FEATURE_CUSTOM_LOGO)
    );

    const [canEdit] = usePermissionCheck(["bb.workspaces.update"]);

    const [localTitle, setLocalTitle] = useState(workspace?.title ?? "");
    const [logoUrl, setLogoUrl] = useState(workspace?.logo ?? "");
    const [loading, setLoading] = useState(false);
    const [dropActive, setDropActive] = useState(false);

    const fileInputRef = useRef<HTMLInputElement>(null);

    const workspaceID = (workspace?.name ?? "").replace(/^workspaces\//, "");

    const isDirty = useCallback(() => {
      if (!workspace) return false;
      const titleChanged =
        localTitle !== (workspace.title ?? "") && localTitle.trim() !== "";
      const logoChanged = logoUrl !== (workspace.logo ?? "");
      return titleChanged || logoChanged;
    }, [localTitle, logoUrl, workspace]);

    const revert = useCallback(() => {
      setLocalTitle(workspace?.title ?? "");
      setLogoUrl(workspace?.logo ?? "");
    }, [workspace]);

    const update = useCallback(async () => {
      if (loading || !workspace) return;
      setLoading(true);
      try {
        const updateMasks: string[] = [];
        const ws = create(WorkspaceSchema, workspace);

        if (
          localTitle &&
          localTitle !== workspace.title &&
          localTitle.trim() !== ""
        ) {
          ws.title = localTitle;
          updateMasks.push("title");
        }
        if (logoUrl !== (workspace.logo ?? "")) {
          ws.logo = logoUrl;
          updateMasks.push("logo");
        }

        await workspaceStore.updateWorkspace(ws, updateMasks);
      } finally {
        setLoading(false);
      }
    }, [loading, workspace, localTitle, logoUrl, workspaceStore]);

    // Re-sync state when workspace loads asynchronously, only if not dirty
    useEffect(() => {
      if (!workspace || isDirty()) return;
      setLocalTitle(workspace.title ?? "");
      setLogoUrl(workspace.logo ?? "");
    }, [workspace, isDirty]);

    useImperativeHandle(ref, () => ({ isDirty, revert, update }));

    // Notify parent of dirty changes
    useEffect(() => {
      onDirtyChange();
    }, [localTitle, logoUrl, onDirtyChange]);

    const validateFile = useCallback(
      (file: File): boolean => {
        const extension = `.${file.name.toLowerCase().split(".").pop()}`;
        if (!SUPPORT_IMAGE_EXTENSIONS.includes(extension)) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("common.file-selector.type-limit", {
              extension: SUPPORT_IMAGE_EXTENSIONS.join(", "),
            }),
          });
          return false;
        }
        if (file.size > MAX_FILE_SIZE_MIB * 1024 * 1024) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("common.file-selector.size-limit", {
              size: MAX_FILE_SIZE_MIB,
            }),
          });
          return false;
        }
        return true;
      },
      [t]
    );

    const handleFileSelect = useCallback(
      async (file: File) => {
        if (!validateFile(file)) return;
        const base64 = await convertFileToBase64(file);
        setLogoUrl(base64);
      },
      [validateFile]
    );

    const handleFileInput = useCallback(
      (e: ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) handleFileSelect(file);
        if (fileInputRef.current) fileInputRef.current.value = "";
      },
      [handleFileSelect]
    );

    const handleDrop = useCallback(
      (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        setDropActive(false);
        if (!canEdit || !hasBrandingFeature) return;
        const file = e.dataTransfer?.files?.[0];
        if (file) handleFileSelect(file);
      },
      [canEdit, hasBrandingFeature, handleFileSelect]
    );

    const uploadDisabled = !canEdit || !hasBrandingFeature;

    return (
      <div id="branding" className="py-6 lg:flex">
        <div className="text-left lg:w-1/4">
          <h1 className="text-2xl font-bold">{title}</h1>
        </div>
        <PermissionGuard permissions={["bb.workspaces.update"]} display="block">
          <div className="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
            {/* Workspace ID */}
            <div>
              <label className="font-medium">
                {t("settings.general.workspace.id")}
              </label>
              <Input value={workspaceID} disabled className="mt-1" />
            </div>

            {/* Workspace Title */}
            <div>
              <label className="font-medium">
                {t("settings.general.workspace.title")}
              </label>
              <Input
                value={localTitle}
                onChange={(e) => setLocalTitle(e.target.value)}
                disabled={!canEdit}
                className="mt-1"
              />
            </div>

            {/* Logo */}
            <div>
              <div className="mb-4 mt-4 lg:mt-0">
                <div className="flex items-center gap-x-2 font-medium">
                  {t("settings.general.workspace.logo")}
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_CUSTOM_LOGO}
                    clickable
                  />
                </div>
                <p className="mb-3 text-sm text-gray-400">
                  {t("settings.general.workspace.logo-aspect")}
                </p>
                <div
                  className={`flex justify-center border-2 border-gray-300 border-dashed rounded-xs relative h-48 transition-all ${
                    dropActive ? "bg-gray-300 opacity-100" : ""
                  } ${uploadDisabled ? "cursor-not-allowed" : "cursor-pointer hover:bg-gray-100"}`}
                  onClick={() => {
                    if (!uploadDisabled) fileInputRef.current?.click();
                  }}
                  onDrop={handleDrop}
                  onDragOver={(e) => e.preventDefault()}
                  onDragEnter={() => setDropActive(true)}
                  onDragLeave={() => setDropActive(false)}
                >
                  {/* Logo preview */}
                  <div
                    className="w-full bg-no-repeat bg-contain bg-center rounded-sm pointer-events-none m-4"
                    style={{
                      backgroundImage: logoUrl ? `url(${logoUrl})` : undefined,
                    }}
                  />
                  {/* Upload overlay */}
                  <div
                    className={`flex flex-col gap-y-1 text-center justify-center items-center absolute top-0 bottom-0 left-0 right-0 ${
                      logoUrl ? "opacity-0 hover:opacity-80 hover:bg-gray-100" : ""
                    }`}
                  >
                    {!logoUrl && (
                      <div className="py-4 text-gray-500">
                        {t("common.no-data")}
                      </div>
                    )}
                    <div className="text-sm text-gray-600 inline-flex pointer-events-none">
                      <span className="relative cursor-pointer rounded-xs font-medium text-indigo-600 hover:text-indigo-500">
                        {t("settings.general.workspace.select-logo")}
                      </span>
                      <p className="pl-1">
                        {t("settings.general.workspace.drag-logo")}
                      </p>
                    </div>
                    <p className="text-xs text-gray-500 pointer-events-none">
                      {t("settings.general.workspace.logo-upload-tip", {
                        extension: SUPPORT_IMAGE_EXTENSIONS.join(", "),
                        size: MAX_FILE_SIZE_MIB,
                      })}
                    </p>
                  </div>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept={SUPPORT_IMAGE_EXTENSIONS.join(",")}
                    className="sr-only hidden"
                    disabled={uploadDisabled}
                    onChange={handleFileInput}
                  />
                </div>
              </div>
              {logoUrl && (
                <div className="flex justify-end gap-x-3">
                  <Button
                    variant="destructive"
                    disabled={!canEdit}
                    onClick={() => setLogoUrl("")}
                  >
                    {t("common.delete")}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </PermissionGuard>
      </div>
    );
  }
);
