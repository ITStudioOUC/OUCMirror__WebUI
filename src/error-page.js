// Bundled as a small independent script: an unavailable API must not break its error page.
export function initializeErrorPage(messages) {
  const read = key => { try { return localStorage.getItem(key) } catch { return null } }
  const save = (key, value) => { try { localStorage.setItem(key, value) } catch { /* Optional preference. */ } }
  const locales = Object.keys(messages)
  const savedLocale = read('mirrorouc.locale')
  const preferred = navigator.languages.map(value => locales.find(locale => locale.toLowerCase() === value.toLowerCase()) || locales.find(locale => locale.split('-')[0] === value.split('-')[0])).find(Boolean)
  let locale = locales.includes(savedLocale) ? savedLocale : preferred || 'zh-CN'
  const storedTheme = read('mirrorouc.theme')
  let theme = ['light', 'dark'].includes(storedTheme) ? storedTheme : 'system'
  const system = matchMedia('(prefers-color-scheme: dark)')
  const language = document.getElementById('error-language')
  const themes = document.getElementById('error-theme')
  const code = document.body.dataset.errorStatus
  function applyTheme() {
    const dark = theme === 'dark' || (theme === 'system' && system.matches)
    document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    document.querySelector('meta[name="theme-color"]').content = dark ? '#0d1117' : '#ffffff'
    themes.value = theme
  }
  function applyLocale() {
    const value = messages[locale]
    const state = value.errors.codes[code]
    document.documentElement.lang = locale
    document.title = `${code} · ${state.title} · ${value.common.brand}`
    document.getElementById('error-title').textContent = state.title
    document.getElementById('error-description').textContent = state.description
    document.getElementById('error-home').textContent = value.errors.home
    document.getElementById('error-retry').textContent = value.errors.retry
    document.querySelector('.error-brand').setAttribute('aria-label', value.common.siteName)
    language.setAttribute('aria-label', value.common.language)
    themes.setAttribute('aria-label', value.common.theme)
    for (const option of themes.options) option.textContent = value.common.themes[option.value]
    language.value = locale
  }
  language.addEventListener('change', () => {
    if (!locales.includes(language.value)) return
    locale = language.value; save('mirrorouc.locale', locale); applyLocale()
  })
  themes.addEventListener('change', () => {
    theme = themes.value; save('mirrorouc.theme', theme); applyTheme()
  })
  system.addEventListener('change', applyTheme)
  document.getElementById('error-retry').addEventListener('click', () => location.reload())
  applyTheme(); applyLocale()
  document.getElementById('error-tools').hidden = false
  document.getElementById('error-retry').hidden = false
}
