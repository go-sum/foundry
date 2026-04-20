(() => {
  var p = localStorage.getItem('themePreference') || 'system';
  document.documentElement.setAttribute('data-theme-preference', p);
  if (p === 'dark' || (p === 'system') && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    document.documentElement.classList.add('dark');
  }
})();
