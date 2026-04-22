import { useEffect, useRef } from "react";
import {
  type SQLEditorEvents,
  sqlEditorEvents,
} from "@/views/sql-editor/events";

export function useSQLEditorEvent<E extends keyof SQLEditorEvents>(
  event: E,
  handler: (data: SQLEditorEvents[E]) => void
): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    const unsubscribe = sqlEditorEvents.on(event, (data) => {
      handlerRef.current(data as SQLEditorEvents[E]);
    });
    return unsubscribe;
  }, [event]);
}
