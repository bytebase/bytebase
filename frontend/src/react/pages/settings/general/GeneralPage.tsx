import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { t as vueT } from "@/plugins/i18n";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { pushNotification, useActuatorV1Store } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";
import { AccountSection } from "./AccountSection";
import { AIAugmentationSection } from "./AIAugmentationSection";
import { AnnouncementSection } from "./AnnouncementSection";
import { AuditLogSection } from "./AuditLogSection";
import { BrandingSection } from "./BrandingSection";
import { GeneralSection } from "./GeneralSection";
import { ProductImprovementSection } from "./ProductImprovementSection";
import { SecuritySection } from "./SecuritySection";
import { SQLEditorSection } from "./SQLEditorSection";
import type { SectionHandle } from "./useSettingSection";

export function GeneralPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const canEditProfile = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );

  const generalRef = useRef<SectionHandle>(null);
  const brandingRef = useRef<SectionHandle>(null);
  const accountRef = useRef<SectionHandle>(null);
  const securityRef = useRef<SectionHandle>(null);
  const sqlEditorRef = useRef<SectionHandle>(null);
  const aiRef = useRef<SectionHandle>(null);
  const announcementRef = useRef<SectionHandle>(null);
  const productImprovementRef = useRef<SectionHandle>(null);
  const auditLogRef = useRef<SectionHandle>(null);

  // Dirty state: sections call onDirtyChange to trigger a re-render,
  // which re-evaluates isDirty from all refs.
  const [, setDirtyKey] = useState(0);
  const onDirtyChange = useCallback(() => setDirtyKey((k) => k + 1), []);

  const allRefs = [
    generalRef,
    brandingRef,
    accountRef,
    securityRef,
    sqlEditorRef,
    aiRef,
    announcementRef,
    productImprovementRef,
    auditLogRef,
  ];

  const isDirty = allRefs.some((ref) => ref.current?.isDirty());

  const handleRevert = useCallback(() => {
    for (const ref of allRefs) {
      ref.current?.revert();
    }
    setDirtyKey((k) => k + 1);
  }, []);

  const handleUpdate = useCallback(async () => {
    const sections: { name: string; handle: SectionHandle }[] = [
      { name: t("common.general"), handle: generalRef.current! },
      {
        name: t("settings.general.workspace.branding"),
        handle: brandingRef.current!,
      },
      {
        name: t("settings.general.workspace.account"),
        handle: accountRef.current!,
      },
      {
        name: t("settings.general.workspace.security"),
        handle: securityRef.current!,
      },
      { name: t("sql-editor.self"), handle: sqlEditorRef.current! },
      {
        name: t("settings.general.workspace.ai-assistant.self"),
        handle: aiRef.current!,
      },
      {
        name: t("settings.general.workspace.announcement.self"),
        handle: announcementRef.current!,
      },
    ];
    if (productImprovementRef.current) {
      sections.push({
        name: t("settings.general.workspace.product-improvement.self"),
        handle: productImprovementRef.current,
      });
    }
    if (auditLogRef.current) {
      sections.push({
        name: t("settings.general.workspace.log"),
        handle: auditLogRef.current,
      });
    }

    let failedCount = 0;
    let totalCount = 0;
    for (const { name, handle } of sections) {
      if (handle.isDirty()) {
        totalCount++;
        try {
          await handle.update();
        } catch (e) {
          console.error(e);
          failedCount++;
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: t("settings.general.workspace.failed-to-update-setting", {
              title: name,
            }),
          });
        }
      }
    }
    if (totalCount > 0 && totalCount !== failedCount) {
      pushNotification({
        module: "bytebase",
        style: failedCount === 0 ? "SUCCESS" : "WARN",
        title:
          failedCount === 0
            ? t("settings.general.workspace.config-updated")
            : t("settings.general.workspace.config-partly-updated"),
      });
    }
    setDirtyKey((k) => k + 1);
  }, [t]);

  // Route change guard — both browser close and in-app navigation
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (allRefs.some((ref) => ref.current?.isDirty())) {
        e.preventDefault();
      }
    };
    window.addEventListener("beforeunload", handleBeforeUnload);

    const removeGuard = router.beforeEach((_to, _from, next) => {
      if (allRefs.some((ref) => ref.current?.isDirty())) {
        if (!window.confirm(vueT("common.leave-without-saving"))) {
          next(false);
          return;
        }
      }
      next();
    });

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
      removeGuard();
    };
  }, []);

  // Hash scroll on mount
  useEffect(() => {
    if (location.hash) {
      document.body.querySelector(location.hash)?.scrollIntoView();
    }
  }, []);

  return (
    <div className="px-4 divide-y divide-block-border py-4">
      <GeneralSection
        ref={generalRef}
        title={t("common.general")}
        onDirtyChange={onDirtyChange}
      />
      <BrandingSection
        ref={brandingRef}
        title={t("settings.general.workspace.branding")}
        onDirtyChange={onDirtyChange}
      />
      <AccountSection
        ref={accountRef}
        title={t("settings.general.workspace.account")}
        onDirtyChange={onDirtyChange}
      />
      <SecuritySection
        ref={securityRef}
        title={t("settings.general.workspace.security")}
        onDirtyChange={onDirtyChange}
      />
      <SQLEditorSection
        ref={sqlEditorRef}
        title={t("sql-editor.self")}
        onDirtyChange={onDirtyChange}
      />
      <AIAugmentationSection
        ref={aiRef}
        title={t("settings.general.workspace.ai-assistant.self")}
        onDirtyChange={onDirtyChange}
      />
      <AnnouncementSection
        ref={announcementRef}
        title={t("settings.general.workspace.announcement.self")}
        onDirtyChange={onDirtyChange}
      />
      {!isSaaSMode && (
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <ProductImprovementSection
            ref={productImprovementRef}
            allowEdit={canEditProfile}
            onDirtyChange={onDirtyChange}
          />
        </PermissionGuard>
      )}
      {!isSaaSMode && (
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <AuditLogSection
            ref={auditLogRef}
            allowEdit={canEditProfile}
            onDirtyChange={onDirtyChange}
          />
        </PermissionGuard>
      )}

      {isDirty && (
        <div className="sticky bottom-0 z-10 -mb-4">
          <div className="flex justify-between w-full py-4 border-t border-block-border bg-white">
            <Button variant="outline" onClick={handleRevert}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleUpdate}>{t("common.update")}</Button>
          </div>
        </div>
      )}
    </div>
  );
}
