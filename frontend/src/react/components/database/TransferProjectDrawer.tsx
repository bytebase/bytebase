import { X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useActuatorV1Store, useProjectV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getInstanceResource,
} from "@/utils";

export function TransferProjectDrawer({
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

  useEscapeKey(open, onClose);

  const fetchProjects = useCallback(
    async (query: string) => {
      setLoadingProjects(true);
      try {
        const { projects: result } = await projectStore.fetchProjectList({
          filter: { query, excludeDefault: true },
          pageSize: 50,
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

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[36rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("database.transfer-project")}
          </h2>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-y-4">
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
                  className="h-4 w-4"
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
                          : "hover:bg-gray-50"
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
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
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
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}
