// a publish-subscribe pattern
import { EventType } from "@/types";

class Event {
  eventList: Partial<Record<EventType, (...args: any[]) => any>>;
  constructor() {
    this.eventList = {};
  }

  on(event: EventType, handler: (...args: any[]) => any) {
    if (!this.eventList[event]) {
      this.eventList[event] = handler;
    }
  }

  off(event: EventType) {
    if (this.eventList[event]) {
      delete this.eventList[event];
    }
  }

  emit(event: EventType, ...params: any[]) {
    if (this.eventList[event]) {
      this.eventList[event]?.(...params);
    }
  }
}

const event = new Event();

export { event, Event };
