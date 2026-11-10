import { Popover as BasePopover } from "@base-ui/react/popover";
import { Building2, Check, ChevronDown, FolderKanban } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { useProjectList } from "@/hooks/useAppState";

interface AgentContextPickerProps {
  context?: string;
  contextTitle?: string;
  workspace?: { name: string; title: string };
  disabled: boolean;
  onSelect: (context: string) => boolean;
}

interface ContextOptionsProps extends AgentContextPickerProps {
  close: () => void;
}

function ContextOptions({
  context,
  workspace,
  onSelect,
  close,
}: ContextOptionsProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const { projects, isLoading, isFetchingMore, hasMore, loadMore } =
    useProjectList(query);

  const select = (resource: string) => {
    if (onSelect(resource)) close();
  };

  return (
    <div className="flex max-h-80 flex-col" data-agent-context-options>
      <Input
        size="sm"
        className="mb-2 shrink-0"
        autoComplete="off"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t("common.filter-by-name")}
        aria-label={t("common.filter-by-name")}
      />
      <div className="min-h-0 overflow-y-auto">
        {workspace?.name ? (
          <div>
            <div className="px-2 py-1 text-xs font-medium text-control-light">
              {t("agent.context-workspace")}
            </div>
            <Button
              appearance="secondary"
              size="sm"
              className="h-auto w-full justify-start gap-2 px-2 py-2 text-left text-sm"
              data-agent-context-option={workspace.name}
              onClick={() => select(workspace.name)}
            >
              <Building2 className="size-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate">
                {workspace.title || workspace.name}
              </span>
              {context === workspace.name ? <Check className="size-4" /> : null}
            </Button>
          </div>
        ) : null}
        <div className="px-2 py-1 text-xs font-medium text-control-light">
          {t("common.projects")}
        </div>
        {projects.map((project) => (
          <Button
            key={project.name}
            appearance="secondary"
            size="sm"
            className="h-auto w-full justify-start gap-2 px-2 py-2 text-left text-sm"
            data-agent-context-option={project.name}
            onClick={() => select(project.name)}
          >
            <FolderKanban className="size-4 shrink-0" />
            <span className="min-w-0 flex-1 text-left">
              <span className="block truncate">
                {project.title || project.name}
              </span>
            </span>
            {context === project.name ? <Check className="size-4" /> : null}
          </Button>
        ))}
        {!isLoading && projects.length === 0 ? (
          <p className="px-2 py-2 text-xs text-control-light">
            {t("agent.context-no-projects")}
          </p>
        ) : null}
        {isLoading ? (
          <p className="px-2 py-2 text-xs text-control-light">
            {t("common.loading")}
          </p>
        ) : null}
        {hasMore ? (
          <Button
            appearance="secondary"
            size="sm"
            className="w-full justify-start"
            disabled={isFetchingMore}
            onClick={loadMore}
          >
            {t("common.load-more")}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

export function AgentContextPicker(props: AgentContextPickerProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (props.disabled) setOpen(false);
  }, [props.disabled]);

  return (
    <BasePopover.Root open={open} onOpenChange={setOpen}>
      <BasePopover.Trigger
        render={
          <Button
            appearance="secondary"
            size="sm"
            className="h-auto min-w-0 max-w-full justify-start px-0 py-0 text-xs text-control-light hover:bg-transparent hover:text-control"
          />
        }
        disabled={props.disabled}
        data-agent-context-trigger
        title={props.context}
      >
        <span className="min-w-0 truncate">
          {t("agent.context")}:{" "}
          <span data-agent-context>
            {props.contextTitle || t("agent.context-select")}
          </span>
        </span>
        <ChevronDown className="size-3 shrink-0" />
      </BasePopover.Trigger>
      <BasePopover.Portal container={getLayerRoot("agent")}>
        <BasePopover.Positioner
          side="top"
          align="start"
          sideOffset={6}
          className={LAYER_SURFACE_CLASS}
        >
          <BasePopover.Popup
            aria-label={t("agent.context-select")}
            className="w-80 max-w-[calc(100vw-2rem)] rounded-sm border border-control-border bg-background p-2 shadow-md focus:outline-hidden"
          >
            {open ? (
              <ContextOptions {...props} close={() => setOpen(false)} />
            ) : null}
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  );
}
