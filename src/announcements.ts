export const announcementAnchor = (id: string) => `announcement-${id}`
export const announcementUrl = (id: string) => `/announcements#${encodeURIComponent(announcementAnchor(id))}`
