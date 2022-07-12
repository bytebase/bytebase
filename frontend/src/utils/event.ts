// a publish-subscribe pattern

class Event {
  eventList: Record<string, (...args: any[]) => any>;
  constructor() {
    this.eventList = {};
  }

  on(event: string, fn: (...args: any[]) => any) {
    if (!this.eventList[event]) {
      this.eventList[event] = fn;
    }
  }

  off(event: string) {
    if (this.eventList[event]) {
      delete this.eventList[event];
    }
  }

  emit(event: string, params?: any) {
    if (this.eventList[event]) {
      this.eventList[event](params);
    }
  }
}

const event = new Event();

export { event, Event };
