import type Emittery from "emittery";
import type {
  DatalessEventNames,
  EventName,
  OmnipresentEventData,
} from "emittery";
import { onUnmounted } from "vue";

export const useEmitteryEventListener = <
  EventData = Record<EventName, unknown>,
  AllEventData = EventData & OmnipresentEventData,
  DatalessEvents = DatalessEventNames<EventData>,
  E extends keyof AllEventData = keyof AllEventData,
>(
  target: Emittery<EventData, AllEventData, DatalessEvents>,
  event: E | readonly E[],
  listener: (eventData: AllEventData[E]) => void | Promise<void>
) => {
  const unsubscribe = target.on(event, listener);

  onUnmounted(() => {
    unsubscribe();
  });
};
