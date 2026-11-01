import commonZh from '../locales/zh-CN/common.ts'
import commonEn from '../locales/en-US/common.ts'
import directoryZh from '../locales/zh-CN/directory.ts'
import directoryEn from '../locales/en-US/directory.ts'

// Shared by the build-time Nginx templates and optional browser enhancements.
export const directoryMessages = {
  'zh-CN': { common: commonZh, directory: directoryZh },
  'en-US': { common: commonEn, directory: directoryEn },
}
