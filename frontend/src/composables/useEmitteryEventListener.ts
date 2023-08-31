import Emittery, {
  DatalessEventNames,
  EventName,
  OmnipresentEventData,
} from "emittery";
import { onUnmounted } from "vue";

export const useEmitteryEventListener = <
  EventData = Record<EventName, any>,
  AllEventData = EventData & OmnipresentEventData,
  DatalessEvents = DatalessEventNames<EventData>,
  E extends keyof AllEventData = any
>(
  target: Emittery<
    EventData, // TODO: Use `unknown` instead of `any`.
    AllEventData,
    DatalessEvents
  >,
  event: E | readonly E[],
  listener: (eventData: AllEventData[E]) => void | Promise<void>
) => {
  const unsubscribe = target.on(event, listener);

  onUnmounted(() => {
    unsubscribe();
  });
};
