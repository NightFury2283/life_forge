"use strict";
const $ = id => document.getElementById(id);
function notice(text) { $("notice").textContent = text; $("notice").hidden = !text; }
function signedIn(value) { $("workspace").hidden = !value; $("logout").hidden = !value; $("login").hidden = value; }
async function request(path, options = {}) {
  const response = await fetch(path, {credentials: "same-origin", ...options});
  if (response.status === 401) { signedIn(false); throw new Error("Войди в аккаунт, чтобы продолжить."); }
  if (!response.ok) {
    const message = response.status === 409 ? "Характеристика с таким названием уже существует." : response.status === 400 ? "Проверь название, описание и начальное значение." : "Не удалось выполнить запрос. Попробуй ещё раз.";
    throw new Error(message);
  }
  return response.status === 204 ? null : response.json();
}
async function refresh() {
  const [profile, stats] = await Promise.all([request("/api/v1/gamification/profile"), request("/api/v1/gamification/stats")]);
  signedIn(true);
  $("level").textContent = $("hero-level").textContent = profile.level;
  $("coins").textContent = profile.coins;
  $("xp").textContent = `${profile.xp} / ${profile.xp_for_next_level} XP`;
  $("xp-bar").style.width = `${Math.min(100, Math.max(0, profile.xp / Math.max(1, profile.xp_for_next_level) * 100))}%`;
  $("stat-count").textContent = stats.length;
  $("stats").replaceChildren();
  if (!stats.length) {
    const empty = document.createElement("div"); empty.className = "empty";
    empty.textContent = "Здесь появятся твои характеристики. Добавь первый навык — пусть это станет началом.";
    $("stats").append(empty);
  }
  for (const stat of stats) {
    const card = document.createElement("article"); card.className = "card stat";
    const icon = document.createElement("div"); icon.className = "stat-icon"; icon.textContent = "✦"; icon.setAttribute("aria-hidden", "true");
    const detail = document.createElement("div"); detail.className = "stat-detail";
    const title = document.createElement("h3"); title.textContent = stat.title; detail.append(title);
    if (stat.description) { const p = document.createElement("p"); p.textContent = stat.description; detail.append(p); }
    const value = document.createElement("strong"); value.textContent = stat.value;
    const remove = document.createElement("button"); remove.className = "delete"; remove.textContent = "×"; remove.setAttribute("aria-label", `Удалить ${stat.title}`);
    remove.onclick = () => perform(remove, async () => { await request(`/api/v1/gamification/stats/${stat.id}`, {method:"DELETE"}); await refresh(); });
    card.append(icon, detail, value, remove); $("stats").append(card);
  }
}
async function perform(button, action) {
  button.disabled = true; notice("");
  try { await action(); } catch (err) { notice(err.message); } finally { button.disabled = false; }
}
$("stat-form").onsubmit = event => {
  event.preventDefault();
  perform($("create"), async () => {
    const description = $("description").value.trim();
    await request("/api/v1/gamification/stats", {method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({title:$("title").value.trim(), description:description || null, value:Number($("value").value)})});
    $("stat-form").reset(); await refresh();
  });
};
$("demo").onclick = () => perform($("demo"), async () => { await request("/auth/demo", {method:"POST"}); await refresh(); });
$("logout").onclick = () => perform($("logout"), async () => { await request("/auth/logout", {method:"POST"}); signedIn(false); $("stats").replaceChildren(); });
$("refresh").onclick = () => perform($("refresh"), refresh);
async function init() {
  try {
    const status = await request("/api/status");
    $("mode").textContent = status.demo ? "ДЕМО · ЛОКАЛЬНО" : "ЛИЧНЫЙ КАБИНЕТ";
    $("demo").hidden = $("demo-note").hidden = !status.demo;
    $("google").hidden = !status.google_login;
    const profile = await fetch("/api/v1/gamification/profile", {credentials:"same-origin"});
    if (profile.status === 401) { signedIn(false); return; }
    if (!profile.ok) { throw new Error("Профиль пока недоступен. Попробуй войти заново."); }
    await refresh();
  } catch (err) { signedIn(false); notice(err.message); }
}
init();
