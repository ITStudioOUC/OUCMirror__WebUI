export const datetimeFormats = {
  monitorTick: { hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZone: 'Asia/Shanghai' },
  monitorSample: { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23', timeZone: 'Asia/Shanghai' },
  timestamp: { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Shanghai' },
  article: { dateStyle: 'long', timeZone: 'Asia/Shanghai' },
  announcement: {
    year: 'numeric', month: '2-digit', day: '2-digit',
    timeZone: 'Asia/Shanghai', numberingSystem: 'latn', calendar: 'gregory',
  },
} satisfies Record<string, Intl.DateTimeFormatOptions>

export const numberFormats = {
  size: { maximumFractionDigits: 1 },
  integer: { maximumFractionDigits: 0, useGrouping: false },
} satisfies Record<string, Intl.NumberFormatOptions>
