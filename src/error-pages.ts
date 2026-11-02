import commonZh from './locales/zh-CN/common.ts'
import commonEn from './locales/en-US/common.ts'
import errorsZh from './locales/zh-CN/errors.ts'
import errorsEn from './locales/en-US/errors.ts'

export const errorMessages = {
  'zh-CN': { common: commonZh, errors: errorsZh },
  'en-US': { common: commonEn, errors: errorsEn },
}
export const errorCodes = Object.keys(errorsZh.codes)
export const apiErrorCodes: Record<string, string> = {
  '400': 'INVALID_INPUT', '401': 'UNAUTHORIZED', '403': 'FORBIDDEN', '404': 'NOT_FOUND',
  '405': 'METHOD_NOT_ALLOWED', '408': 'REQUEST_TIMEOUT', '410': 'NOT_FOUND',
  '413': 'PAYLOAD_TOO_LARGE', '414': 'INVALID_INPUT', '429': 'RATE_LIMITED',
  '500': 'INTERNAL_ERROR', '502': 'UPSTREAM_UNAVAILABLE', '503': 'UPSTREAM_UNAVAILABLE', '504': 'UPSTREAM_UNAVAILABLE',
}
