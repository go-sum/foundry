(() => {
  var btn = document.querySelector('[data-theme-selector]');
  if (!btn) return;
  btn.addEventListener('click', () => {
    var themes = ['light', 'dark', 'system'];
    var cur = localStorage.getItem('themePreference') || 'system';
    var next = themes[(themes.indexOf(cur) + 1) % themes.length];
    localStorage.setItem('themePreference', next);
    document.documentElement.setAttribute('data-theme-preference', next);
    if (next === 'dark' || (next === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  });
})();
