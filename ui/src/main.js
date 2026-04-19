import { invoke } from "@tauri-apps/api/core";
import DOMPurify from "dompurify";
import "./style.css";

// ─── DOM helpers ─────────────────────────────────────────────
function el(tag, opts = {}, children = []) {
  const e = document.createElement(tag);
  if (opts.className) e.className = opts.className;
  if (opts.id) e.id = opts.id;
  if (opts.text !== undefined) e.textContent = opts.text;
  if (opts.html !== undefined) e.innerHTML = DOMPurify.sanitize(opts.html);
  if (opts.attrs) for (const [k, v] of Object.entries(opts.attrs)) e.setAttribute(k, v);
  if (opts.on) for (const [evt, fn] of Object.entries(opts.on)) e.addEventListener(evt, fn);
  for (const c of children) if (c) e.appendChild(c);
  return e;
}

function safeHtml(htmlString) {
  return DOMPurify.sanitize(htmlString);
}

// ─── State ───────────────────────────────────────────────────
const state = {
  sources: [],
  activeSource: null,
  entries: [],
  filter: "",
  selected: null,
  describe: null,
  helpText: null,
  revealed: {},
  cmd: { text: "hort status", action: false },
  modal: null,
  toast: null,
};

// ─── Tauri bridge ────────────────────────────────────────────
async function d(method, params = {}) {
  try {
    const r = await invoke("hort_rpc", { method, params });
    return { ok: true, result: r };
  } catch (e) {
    return { ok: false, error: String(e) };
  }
}

async function shell(args, passphrase = null) {
  try {
    const r = await invoke("hort_shell", { args, passphrase });
    return { ok: true, result: r };
  } catch (e) {
    return { ok: false, error: String(e) };
  }
}

async function copyToClipboard(text) {
  try { await invoke("hort_copy", { text }); } catch {}
}

// ─── Command-string formatting ──────────────────────────────
function cmdFor(kind, p = {}) {
  const src = p.source ? ` --source ${p.source}` : "";
  const env = p.env && p.env !== "*" ? ` --env ${p.env}` : "";
  const ctx = p.context && p.context !== "*" ? ` --context ${p.context}` : "";
  switch (kind) {
    case "status":         return "hort status";
    case "source_list":    return "hort source list";
    case "list":           return `hort --list${p.source ? " --source " + p.source : ""}`;
    case "describe":       return `hort --describe ${p.name}`;
    case "get_secret":     return `hort --secret ${p.name}${env}${ctx}${src}`;
    case "get_config":     return `hort --config ${p.name}${env}${ctx}${src}`;
    case "set_secret":     return `hort --set-secret ${p.name} --value '${p.value ?? "…"}'${env}${ctx}${p.description ? ` --description '${p.description}'` : ""}${src}`;
    case "set_config":     return `hort --set-config ${p.name} --value '${p.value ?? "…"}'${env}${ctx}${p.description ? ` --description '${p.description}'` : ""}${src}`;
    case "delete":         return `hort --delete ${p.name}${env}${ctx}${src}`;
    case "unlock":         return `hort unlock`;
    case "lock":           return `hort lock`;
    case "daemon_start":   return `hort daemon start`;
    case "source_mount":   return `hort source mount --name ${p.name} --path ${p.path} --key-hex ${p.keyHex ? "…" : "<32-byte-hex>"}`;
    case "source_unmount": return `hort source unmount --name ${p.name}`;
    default:               return "hort --help";
  }
}

function renderCmdHtml(text) {
  return text
    .replace(/^(\S+)/, '<span class="arg">$1</span>')
    .replace(/ (--?\S+)/g, ' <span class="flag">$1</span>')
    .replace(/'([^']*)'/g, `'<span class="val">$1</span>'`);
}

function setViewCmd(kind, p = {}) {
  state.cmd = { text: cmdFor(kind, p), action: false };
  renderCmdStrip();
}

// ─── Loaders ────────────────────────────────────────────────
async function loadSources() {
  const r = await d("status");
  if (!r.ok) { toast(r.error, "err"); return; }
  state.sources = r.result.sources || [];
}

async function loadEntries() {
  const r = await d("list", {});
  if (!r.ok) { return; }
  state.entries = r.result.entries || [];
}

async function loadHelp() {
  if (state.helpText) return;
  const r = await d("help", {});
  if (r.ok && r.result && r.result.text) {
    state.helpText = r.result.text;
    renderDetail();
  }
}

function parseHelp(text) {
  const lines = text.split("\n");
  // Drop trailing empty lines
  while (lines.length && !lines[lines.length - 1].trim()) lines.pop();

  let tagline = "";
  if (lines.length && lines[0].trim()) {
    tagline = lines.shift().trim();
    while (lines.length && !lines[0].trim()) lines.shift();
  }

  const sections = [];
  let current = null;
  for (const raw of lines) {
    const isHeader = raw.length > 0 && raw[0] !== " " && raw[0] !== "\t" && raw.trimEnd().endsWith(":");
    if (isHeader) {
      if (current) sections.push(current);
      current = { title: raw.replace(/:\s*$/, ""), lines: [] };
      continue;
    }
    if (!current) continue;
    current.lines.push(raw.replace(/^  /, ""));
  }
  if (current) sections.push(current);

  // Trim leading/trailing blanks from each section body
  for (const s of sections) {
    while (s.lines.length && !s.lines[0].trim()) s.lines.shift();
    while (s.lines.length && !s.lines[s.lines.length - 1].trim()) s.lines.pop();
  }
  return { tagline, sections };
}

function renderHelpDoc() {
  const wrap = el("div", { className: "help-doc" });
  if (!state.helpText) {
    loadHelp();
    wrap.appendChild(el("div", { className: "help-loading", text: "loading help…" }));
    return wrap;
  }
  const { tagline, sections } = parseHelp(state.helpText);
  if (tagline) {
    wrap.appendChild(el("h1", { className: "help-title", text: "hort" }));
    wrap.appendChild(el("p", { className: "help-tagline", text: tagline.replace(/^hort\s*—\s*/i, "") }));
  }
  for (const s of sections) {
    const block = el("section", { className: "help-section" });
    block.appendChild(el("h2", { className: "help-section-title", text: s.title }));
    const pre = el("pre", { className: "help-section-body" });
    pre.textContent = s.lines.join("\n");
    block.appendChild(pre);
    wrap.appendChild(block);
  }
  return wrap;
}

async function loadDescribe(name) {
  const r = await d("describe", { name });
  if (!r.ok) { toast(r.error, "err"); state.describe = null; return; }
  state.describe = r.result.entries || [];
}

// ─── Renderers ──────────────────────────────────────────────
const app = document.getElementById("app");

function render() {
  app.replaceChildren();
  app.appendChild(renderTopbar());
  app.appendChild(renderMain());
  app.appendChild(renderCmdStripEl());
  if (state.modal) app.appendChild(state.modal);
  if (state.toast) app.appendChild(state.toast);
}

function renderTopbar() {
  const brand = el("div", { className: "brand" }, [
    el("span", { className: "key", text: "🔑" }),
    el("span", { text: "hort" }),
  ]);

  const strip = el("div", { className: "sources-strip" });
  for (const s of state.sources) {
    const chip = el("button", {
      className: "source-chip"
        + (s.primary ? " primary" : "")
        + (s.unlocked ? "" : " locked")
        + (state.activeSource === s.name ? " active" : ""),
      on: {
        click: () => {
          state.activeSource = state.activeSource === s.name ? null : s.name;
          setViewCmd("list", { source: state.activeSource });
          render();
        },
      },
    }, [
      el("span", { text: s.name }),
    ]);
    strip.appendChild(chip);
  }

  const actions = el("div", { className: "topbar-actions" });
  const primary = state.sources.find(s => s.primary);
  const primaryLocked = primary && !primary.unlocked;
  if (primaryLocked) {
    actions.appendChild(iconBtn("unlock", () => openUnlockModal()));
  } else if (primary) {
    actions.appendChild(iconBtn("lock", async () => {
      setViewCmd("lock");
      const r = await shell(["lock"]);
      if (!r.ok) toast(r.error, "err");
      else { await refresh(); toast("vault locked"); }
    }));
  }
  actions.appendChild(iconBtn("+ mount", () => openMountModal()));
  actions.appendChild(iconBtn("+ entry", () => openSetEntryModal()));
  actions.appendChild(iconBtn("↻", refresh));

  return el("div", { className: "topbar" }, [brand, strip, actions]);
}

function iconBtn(label, onClick) {
  return el("button", { className: "icon-btn", text: label, on: { click: onClick } });
}

function renderMain() {
  const head = el("div", { className: "section-head" }, [
    el("span", { text: "Entries" + (state.activeSource ? " · " + state.activeSource : "") }),
    el("span", { text: String(filteredEntries().length) }),
  ]);

  const filter = el("input", { className: "filter-input", attrs: { type: "text", placeholder: "filter…", value: state.filter } });
  filter.value = state.filter;
  filter.oninput = (e) => { state.filter = e.target.value; renderEntriesList(); };

  const list = el("div", { className: "entries-list" });
  const entries = el("div", { className: "entries" }, [head, filter, list]);

  const detail = el("div", { className: "detail", id: "detail" });
  const main = el("div", { className: "main" }, [entries, detail]);
  setTimeout(() => { renderEntriesList(); renderDetail(); }, 0);
  return main;
}

function filteredEntries() {
  return state.entries
    .filter(e => {
      if (state.activeSource && e.source !== state.activeSource) return false;
      if (state.filter && !e.name.toLowerCase().includes(state.filter.toLowerCase())) return false;
      return true;
    })
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
}

function renderEntriesList() {
  const container = document.querySelector(".entries-list");
  if (!container) return;
  container.replaceChildren();
  const items = filteredEntries();
  if (!items.length) {
    container.appendChild(el("div", {
      className: "empty-state",
      text: state.sources.some(s => !s.unlocked && s.primary) ? "unlock vault to see entries" : "no entries",
    }));
    return;
  }
  for (const e of items) {
    const isSel = state.selected && state.selected.name === e.name && state.selected.source === e.source;
    const row = el("div", {
      className: "entry-row" + (isSel ? " selected" : ""),
      on: { click: () => selectEntry(e) },
    }, [
      el("span", { className: "entry-name", text: e.name }),
      el("span", { className: "entry-type " + e.type, text: e.type }),
      el("span", { className: "entry-count", text: String((e.variants && e.variants.length) || 1) }),
    ]);
    container.appendChild(row);
  }
}

async function selectEntry(e) {
  state.selected = { source: e.source, name: e.name, type: e.type };
  setViewCmd("describe", { name: e.name });
  await loadDescribe(e.name);
  renderEntriesList();
  renderDetail();
}

function renderDetail() {
  const container = document.getElementById("detail");
  if (!container) return;
  container.replaceChildren();

  if (!state.selected) {
    container.appendChild(renderHelpDoc());
    return;
  }
  if (!state.describe || !state.describe.length) {
    container.appendChild(el("div", { className: "detail-empty", text: "loading…" }));
    return;
  }

  for (const block of state.describe) {
    const head = el("div", {}, [
      el("h1", { className: "detail-name", text: block.name }),
      el("div", { className: "detail-sub" }, [
        el("span", { text: block.type }),
        document.createTextNode(" · "),
        el("span", { className: "src", text: block.source }),
      ]),
    ]);
    container.appendChild(head);

    container.appendChild(el("div", {
      className: "detail-desc" + (block.description ? "" : " empty"),
      text: block.description || "no description",
    }));

    container.appendChild(el("div", { className: "section-head" }, [
      el("span", { text: "Variants · env × context" }),
      el("span", { text: String(block.variants.length) }),
    ]));

    const variants = el("div", { className: "variants" });
    for (const v of block.variants) {
      variants.appendChild(renderVariantRow(block, v));
    }
    container.appendChild(variants);

    const footer = el("div", { className: "detail-footer" }, [
      el("button", {
        className: "primary-btn",
        text: "+ add variant",
        on: { click: () => openSetEntryModal({
          name: block.name, type: block.type, source: block.source, description: block.description,
        }) },
      }),
    ]);
    container.appendChild(footer);
  }
}

function renderVariantRow(block, v) {
  const key = `${block.source}|${block.name}|${v.env}|${v.context}`;
  const isRevealed = !!state.revealed[key];
  const masked = block.type === "secret" && !isRevealed;
  const displayVal = masked ? "••••••••••••" : (v.value ?? "—");

  const actions = el("div", { className: "variant-actions" });

  if (block.type === "secret") {
    const revealBtn = el("button", {
      className: "ghost-btn",
      text: isRevealed ? "hide" : "reveal",
      on: {
        click: async (ev) => {
          ev.stopPropagation();
          if (isRevealed) {
            delete state.revealed[key];
            setViewCmd("describe", { name: block.name });
            renderDetail();
            return;
          }
          const kind = "get_secret";
          setViewCmd(kind, { name: block.name, env: v.env, context: v.context, source: block.source });
          const r = await d(kind, { name: block.name, env: v.env, context: v.context, source: block.source });
          if (!r.ok) { toast(r.error, "err"); return; }
          v.value = r.result.value;
          state.revealed[key] = true;
          renderDetail();
        },
      },
    });
    actions.appendChild(revealBtn);
  }

  const copyBtn = el("button", {
    className: "ghost-btn",
    text: "copy",
    on: {
      click: async (ev) => {
        ev.stopPropagation();
        if (v.value === undefined || v.value === null) {
          const kind = block.type === "secret" ? "get_secret" : "get_config";
          setViewCmd(kind, { name: block.name, env: v.env, context: v.context, source: block.source });
          const r = await d(kind, { name: block.name, env: v.env, context: v.context, source: block.source });
          if (!r.ok) { toast(r.error, "err"); return; }
          v.value = r.result.value;
        }
        await copyToClipboard(v.value);
        flash(copyBtn);
        toast("value copied");
      },
    },
  });
  actions.appendChild(copyBtn);

  const editBtn = el("button", {
    className: "ghost-btn",
    text: "edit",
    on: {
      click: (ev) => {
        ev.stopPropagation();
        openSetEntryModal({
          name: block.name, type: block.type, source: block.source,
          env: v.env, context: v.context, description: block.description,
        });
      },
    },
  });
  actions.appendChild(editBtn);

  const delBtn = el("button", {
    className: "ghost-btn danger",
    text: "del",
    on: {
      click: async (ev) => {
        ev.stopPropagation();
        const cmd = cmdFor("delete", { name: block.name, env: v.env, context: v.context, source: block.source });
        if (!confirm(`Delete this variant?\n\n$ ${cmd}`)) return;
        state.cmd = { text: cmd, action: true };
        renderCmdStrip();
        const r = await d("delete", { name: block.name, env: v.env, context: v.context, source: block.source });
        if (!r.ok) { toast(r.error, "err"); return; }
        toast("deleted");
        await refresh();
        await loadDescribe(block.name);
        renderDetail();
      },
    },
  });
  actions.appendChild(delBtn);

  return el("div", { className: "variant-row" }, [
    el("span", { className: "variant-env" + (v.env === "*" ? " wild" : ""), text: v.env }),
    el("span", { className: "variant-ctx" + (v.context === "*" ? " wild" : ""), text: v.context }),
    el("span", { className: "variant-value" + (masked ? " masked" : ""), text: displayVal }),
    actions,
  ]);
}

// ─── CMD strip ──────────────────────────────────────────────
function renderCmdStripEl() {
  const cmdSpan = el("span", { className: "cmd", html: renderCmdHtml(state.cmd.text) });
  const copyBtn = el("button", { className: "copy-btn", text: "copy", on: {
    click: async () => { await copyToClipboard(state.cmd.text); flash(copyBtn); },
  }});
  return el("div", {
    className: "cmd-strip" + (state.cmd.action ? " action" : ""),
    id: "cmd-strip",
  }, [
    el("span", { className: "prompt", text: "$" }),
    cmdSpan,
    copyBtn,
  ]);
}

function renderCmdStrip() {
  const existing = document.getElementById("cmd-strip");
  if (!existing) return;
  existing.replaceWith(renderCmdStripEl());
}

// ─── Modals ────────────────────────────────────────────────
function buildModalShell(title, bodyChildren, onOk, okLabel = "save") {
  const errEl = el("div", { className: "error" });
  const actions = el("div", { className: "actions" }, [
    el("button", { className: "ghost-btn", text: "cancel", on: { click: () => closeModal() } }),
    el("button", { className: "primary-btn", text: okLabel, on: { click: () => onOk(errEl) } }),
  ]);
  const modal = el("div", { className: "modal" }, [
    el("h2", { text: title }),
    ...bodyChildren,
    errEl,
    actions,
  ]);
  return el("div", { className: "modal-backdrop" }, [modal]);
}

function openUnlockModal() {
  const pw = el("input", { attrs: { type: "password", placeholder: "passphrase" } });
  const preview = el("div", { className: "cmd-preview", text: "$ HORT_PASSPHRASE='…' hort unlock" });
  const m = buildModalShell("Unlock primary vault", [
    el("div", { className: "field" }, [el("label", { text: "Passphrase" }), pw]),
    preview,
  ], async (errEl) => {
    const val = pw.value;
    if (!val) return;
    state.cmd = { text: "hort unlock", action: true };
    renderCmdStrip();
    const r = await shell(["unlock"], val);
    if (!r.ok) { errEl.textContent = r.error; return; }
    closeModal();
    toast("vault unlocked");
    await refresh();
  }, "unlock");
  pw.addEventListener("keydown", (e) => { if (e.key === "Enter") m.querySelector(".primary-btn").click(); });
  state.modal = m;
  render();
  setTimeout(() => pw.focus(), 30);
}

function openMountModal() {
  const nameInput = el("input", { attrs: { type: "text", placeholder: "heine" } });
  const pathInput = el("input", { attrs: { type: "text", placeholder: "/path/to/vault.enc" } });
  const keyInput = el("input", { attrs: { type: "text", placeholder: "64-char hex (32 bytes)" } });
  const preview = el("div", { className: "cmd-preview", text: "$ hort source mount --name … --path … --key-hex …" });
  const update = () => {
    const n = nameInput.value || "…";
    const p = pathInput.value || "…";
    const k = keyInput.value ? "…" : "…";
    preview.textContent = `$ hort source mount --name ${n} --path ${p} --key-hex ${k}`;
  };
  [nameInput, pathInput, keyInput].forEach(i => i.addEventListener("input", update));

  const m = buildModalShell("Mount a source", [
    el("div", { className: "field" }, [el("label", { text: "Name" }), nameInput]),
    el("div", { className: "field" }, [el("label", { text: "Path" }), pathInput]),
    el("div", { className: "field" }, [el("label", { text: "Key (hex)" }), keyInput]),
    preview,
  ], async (errEl) => {
    const name = nameInput.value.trim();
    const path = pathInput.value.trim();
    const key = keyInput.value.trim();
    if (!name || !path || !key) { errEl.textContent = "name, path, key required"; return; }
    state.cmd = { text: cmdFor("source_mount", { name, path, keyHex: key }), action: true };
    renderCmdStrip();
    const r = await shell(["source", "mount", "--name", name, "--path", path, "--key-hex", key]);
    if (!r.ok) { errEl.textContent = r.error; return; }
    closeModal();
    toast("source mounted");
    await refresh();
  }, "mount");
  state.modal = m;
  render();
  setTimeout(() => nameInput.focus(), 30);
}

function openSetEntryModal(prefill = {}) {
  const isEdit = !!prefill.name;
  const primarySrc = state.sources.find(s => s.primary)?.name || "primary";

  const typeSel = el("select");
  for (const t of ["secret", "config"]) {
    const opt = el("option", { attrs: { value: t }, text: t });
    if (prefill.type ? prefill.type === t : t === "secret") opt.selected = true;
    typeSel.appendChild(opt);
  }

  const srcSel = el("select");
  for (const s of state.sources) {
    const opt = el("option", { attrs: { value: s.name }, text: s.name });
    if (prefill.source ? prefill.source === s.name : s.name === primarySrc) opt.selected = true;
    srcSel.appendChild(opt);
  }

  const nameInput = el("input", { attrs: { type: "text", placeholder: "api-key", value: prefill.name || "" } });
  if (isEdit) nameInput.readOnly = true;
  nameInput.value = prefill.name || "";

  const envInput = el("input", { attrs: { type: "text", placeholder: "prod" } });
  envInput.value = prefill.env && prefill.env !== "*" ? prefill.env : "";
  const ctxInput = el("input", { attrs: { type: "text", placeholder: "heine" } });
  ctxInput.value = prefill.context && prefill.context !== "*" ? prefill.context : "";
  const valInput = el("textarea", { attrs: { placeholder: "secret or config value" } });
  const descInput = el("input", { attrs: { type: "text", value: prefill.description || "" } });
  descInput.value = prefill.description || "";

  const preview = el("div", { className: "cmd-preview" });

  const update = () => {
    const type = typeSel.value;
    const name = nameInput.value || "…";
    const env = envInput.value || "*";
    const ctx = ctxInput.value || "*";
    const desc = descInput.value;
    const src = srcSel.value;
    preview.textContent = "$ " + cmdFor(type === "secret" ? "set_secret" : "set_config", {
      name, env, context: ctx, description: desc, source: src,
    });
  };
  [typeSel, srcSel, nameInput, envInput, ctxInput, valInput, descInput].forEach(i => {
    i.addEventListener("input", update);
    i.addEventListener("change", update);
  });
  update();

  const body = [
    el("div", { className: "row" }, [
      el("div", { className: "field" }, [el("label", { text: "Type" }), typeSel]),
      el("div", { className: "field" }, [el("label", { text: "Source" }), srcSel]),
    ]),
    el("div", { className: "field" }, [el("label", { text: "Name" }), nameInput]),
    el("div", { className: "row" }, [
      el("div", { className: "field" }, [el("label", { text: "Env (optional)" }), envInput]),
      el("div", { className: "field" }, [el("label", { text: "Context (optional)" }), ctxInput]),
    ]),
    el("div", { className: "field" }, [el("label", { text: "Value" }), valInput]),
    el("div", { className: "field" }, [el("label", { text: "Description (optional)" }), descInput]),
    preview,
  ];

  const m = buildModalShell(isEdit ? "Set variant" : "New entry", body, async (errEl) => {
    const type = typeSel.value;
    const name = nameInput.value.trim();
    const value = valInput.value;
    const env = envInput.value.trim();
    const ctx = ctxInput.value.trim();
    const desc = descInput.value.trim();
    const src = srcSel.value;
    if (!name || value === "") { errEl.textContent = "name and value required"; return; }
    const method = type === "secret" ? "set_secret" : "set_config";
    state.cmd = { text: cmdFor(method, { name, env, context: ctx, description: desc, source: src, value: "•••" }), action: true };
    renderCmdStrip();
    const r = await d(method, { name, value, env, context: ctx, description: desc, source: src });
    if (!r.ok) { errEl.textContent = r.error; return; }
    closeModal();
    toast("saved");
    await refresh();
    if (state.selected && state.selected.name === name) {
      await loadDescribe(name);
      renderDetail();
    }
  }, "save");
  state.modal = m;
  render();
  setTimeout(() => (isEdit ? valInput : nameInput).focus(), 30);
}

function closeModal() {
  state.modal = null;
  render();
}

// ─── Toasts ────────────────────────────────────────────────
function toast(msg, kind = "") {
  const t = el("div", { className: "toast " + kind, text: msg });
  state.toast = t;
  render();
  setTimeout(() => t.classList.add("show"), 20);
  setTimeout(() => { state.toast = null; render(); }, 2400);
}

function flash(btn) {
  btn.classList.add("flashed");
  setTimeout(() => btn.classList.remove("flashed"), 600);
}

// ─── Lifecycle ─────────────────────────────────────────────
async function refresh() {
  await loadSources();
  await loadEntries();
  render();
}

async function boot() {
  render();
  const ping = await d("status");
  if (!ping.ok) {
    toast("starting daemon…");
    await shell(["daemon", "start", "--background"]).catch(() => {});
    await new Promise(r => setTimeout(r, 500));
  }
  setViewCmd("source_list");
  await refresh();
}

boot();
