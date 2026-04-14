import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useActuatorV1Store, useProjectV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDefaultPagination,
  getInstanceResource,
} from "@/utils";

export function TransferProjectSheet({
  open,
  databases,
  onClose,
  onTransfer,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onTransfer: (projectName: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const actuatorStore = useActuatorV1Store();
  const defaultProjectName = useVueState(
    () => actuatorStore.serverInfo?.defaultProject ?? ""
  );
  const [mode, setMode] = useState<"project" | "unassign">("project");
  const [searchQuery, setSearchQuery] = useState("");
  const [projects, setProjects] = useState<{ name: string; title: string }[]>(
    []
  );
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [selectedProject, setSelectedProject] = useState("");
  const [transferring, setTransferring] = useState(false);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchProjects = useCallback(
    async (query: string) => {
      setLoadingProjects(true);
      try {
        const { projects: result } = await projectStore.fetchProjectList({
          filter: { query, excludeDefault: true },
          pageSize: getDefaultPagination(),
        });
        setProjects(result.map((p) => ({ name: p.name, title: p.title })));
      } finally {
        setLoadingProjects(false);
      }
    },
    [projectStore]
  );

  useEffect(() => {
    if (open) {
      setMode("project");
      setSearchQuery("");
      setSelectedProject("");
      setTransferring(false);
      fetchProjects("");
    }
  }, [open, fetchProjects]);

  useEffect(() => {
    if (!open) return;
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => fetchProjects(searchQuery), 300);
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, [searchQuery, open, fetchProjects]);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("database.transfer-project")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          <p className="text-sm text-control-light">
            {t("database.selected-n-databases", { n: databases.length })}
          </p>

          <div className="border border-control-border rounded-sm max-h-48 overflow-y-auto">
            {databases.map((db) => (
              <div
                key={db.name}
                className="px-3 py-2 text-sm border-b last:border-b-0 flex items-center gap-x-2"
              >
                <img
                  className="size-4"
                  src={EngineIconPath[getInstanceResource(db).engine]}
                  alt=""
                />
                <span>{extractDatabaseResourceName(db.name).databaseName}</span>
              </div>
            ))}
          </div>

          <div className="flex items-center gap-x-6">
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "project"}
                onChange={() => setMode("project")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">{t("common.project")}</span>
            </label>
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "unassign"}
                onChange={() => setMode("unassign")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">
                {t("database.unassign")}
              </span>
            </label>
          </div>

          {mode === "project" && (
            <div>
              <Input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t("common.filter-by-name")}
                className="mb-2"
              />
              <div className="border border-control-border rounded-sm max-h-64 overflow-y-auto">
                {loadingProjects ? (
                  <div className="px-3 py-4 text-sm text-center text-control-placeholder">
                    {t("common.loading")}
                  </div>
                ) : projects.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-center text-control-placeholder">
                    {t("common.no-data")}
                  </div>
                ) : (
                  projects.map((project) => (
                    <label
                      key={project.name}
                      className={cn(
                        "flex items-center gap-x-3 px-3 py-2.5 cursor-pointer border-b last:border-b-0 transition-colors",
                        selectedProject === project.name
                          ? "bg-accent/5"
                          : "hover:bg-control-bg"
                      )}
                    >
                      <input
                        type="radio"
                        name="transfer-project"
                        checked={selectedProject === project.name}
                        onChange={() => setSelectedProject(project.name)}
                        className="accent-accent"
                      />
                      <span className="text-sm">{project.title}</span>
                      <span className="text-xs text-control-placeholder">
                        {extractProjectResourceName(project.name)}
                      </span>
                    </label>
                  ))
                )}
              </div>
            </div>
          )}
        </SheetBody>
        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={(mode === "project" && !selectedProject) || transferring}
            onClick={async () => {
              setTransferring(true);
              try {
                const target =
                  mode === "unassign" ? defaultProjectName : selectedProject;
                await onTransfer(target);
                onClose();
              } finally {
                setTransferring(false);
              }
            }}
          >
            {t("common.transfer")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
