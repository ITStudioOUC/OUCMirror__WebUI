// Run before the stylesheet so the directory shares the main site's preference.
try {
  var preference = localStorage.getItem('mirrorouc.theme');
  document.documentElement.dataset.theme = preference === 'dark' || (preference !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light';
} catch (_) {
  document.documentElement.dataset.theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}
