// applyTheme toggles the `light` class on <html>, which overrides the dark CSS palette.
export function applyTheme(theme) {
  const el = document.documentElement
  if (theme === 'light') el.classList.add('light')
  else el.classList.remove('light')
}
