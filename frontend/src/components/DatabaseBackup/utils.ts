export function localToUTC(hour: number, dayOfWeek: number) {
  return alignUTC(hour, dayOfWeek, new Date().getTimezoneOffset() * 60);
}

export function localFromUTC(hour: number, dayOfWeek: number) {
  return alignUTC(hour, dayOfWeek, -new Date().getTimezoneOffset() * 60);
}

export function alignUTC(
  hour: number,
  dayOfWeek: number,
  offsetInSecond: number
) {
  if (hour != -1) {
    hour = hour + offsetInSecond / 60 / 60;
    let dayOffset = 0;
    if (hour > 23) {
      hour = hour - 24;
      dayOffset = 1;
    }
    if (hour < 0) {
      hour = hour + 24;
      dayOffset = -1;
    }
    if (dayOfWeek != -1) {
      dayOfWeek = (7 + dayOfWeek + dayOffset) % 7;
    }
  }
  return { hour, dayOfWeek };
}
