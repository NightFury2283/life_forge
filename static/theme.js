"use strict";
(() => {
  const storageKey = "lifeforge.theme";
  let selected = "dark";
  try { if (localStorage.getItem(storageKey) === "light") selected = "light"; } catch {}
  const apply = (theme, save) => {
    selected = theme === "light" ? "light" : "dark";
    document.documentElement.dataset.theme = selected;
    document.documentElement.style.colorScheme = selected;
    document.querySelectorAll("[data-theme-choice]").forEach(button => {
      button.setAttribute("aria-pressed", String(button.dataset.themeChoice === selected));
    });
    if (save) { try { localStorage.setItem(storageKey, selected); } catch {} }
  };
  // Run before stylesheets load so a saved light theme does not flash dark.
  apply(selected, false);
  document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll("[data-theme-choice]").forEach(button => {
      button.addEventListener("click", () => apply(button.dataset.themeChoice, true));
    });
    apply(selected, false);
  });
})();
