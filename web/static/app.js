/* ── Clock ─────────────────────────────────────────────── */
(function () {
  var el = document.getElementById('clock');
  function tick() {
    if (!el) return;
    var d = new Date();
    el.textContent =
      String(d.getHours()).padStart(2, '0') + ':' +
      String(d.getMinutes()).padStart(2, '0');
  }
  tick();
  setInterval(tick, 1000);
})();

/* ── Search engine preference ───────────────────────────── */
(function () {
  var KEY = 'preferred-search-engine';
  var sel = document.getElementById('search-engine-select');
  if (!sel) return;

  var saved = localStorage.getItem(KEY);
  if (!saved) {
    saved = 'google';
    localStorage.setItem(KEY, saved);
  }
  sel.value = saved;

  sel.addEventListener('change', function () {
    localStorage.setItem(KEY, sel.value);
  });
})();

/* ── Add-link modal ─────────────────────────────────────── */
function openAddModal() {
  document.getElementById('add-modal').classList.add('open');
}

function closeAddModal() {
  document.getElementById('add-modal').classList.remove('open');
}

// Close modals after HTMX swaps the link grid (successful add or edit).
document.addEventListener('htmx:afterSwap', function (e) {
  if (e.detail.target && e.detail.target.id === 'link-grid') {
    closeAddModal();
    closeEditModal();
  }
});

// Close modals when clicking their own backdrop.
document.getElementById('add-modal').addEventListener('click', function (e) {
  if (e.target === this) closeAddModal();
});
document.getElementById('edit-modal').addEventListener('click', function (e) {
  if (e.target === this) closeEditModal();
});

/* ── Edit-link modal ────────────────────────────────────── */
function openEditModal(id, name, href, img) {
  document.getElementById('edit-name').value = name;
  document.getElementById('edit-href').value = href;
  document.getElementById('edit-img').value  = img || '';

  var form = document.getElementById('edit-form');
  form.setAttribute('hx-put', '/links/' + id);
  htmx.process(form); // re-process so HTMX picks up the updated hx-put URL

  document.getElementById('edit-modal').classList.add('open');
}

function closeEditModal() {
  document.getElementById('edit-modal').classList.remove('open');
}

/* ── Settings ───────────────────────────────────────────── */
(function () {
  var KEY = 'holetab-settings';
  var DEFAULTS = { autofocusSearch: false, openLinksNewTab: true };

  function load() {
    try {
      var s = localStorage.getItem(KEY);
      return s ? Object.assign({}, DEFAULTS, JSON.parse(s)) : Object.assign({}, DEFAULTS);
    } catch (e) { return Object.assign({}, DEFAULTS); }
  }

  function save(s) { localStorage.setItem(KEY, JSON.stringify(s)); }

  function applyLinkTargets(newTab) {
    document.querySelectorAll('.link-anchor').forEach(function (a) {
      if (newTab) { a.setAttribute('target', '_blank'); }
      else { a.removeAttribute('target'); }
    });
  }

  var settings = load();

  if (settings.autofocusSearch) {
    var inp = document.querySelector('.search-input');
    if (inp) inp.focus();
  }
  applyLinkTargets(settings.openLinksNewTab);

  window.openSettingsModal = function () {
    document.getElementById('setting-autofocus').checked = settings.autofocusSearch;
    document.getElementById('setting-new-tab').checked   = settings.openLinksNewTab;
    document.getElementById('settings-modal').classList.add('open');
  };

  window.closeSettingsModal = function () {
    document.getElementById('settings-modal').classList.remove('open');
  };

  document.getElementById('settings-modal').addEventListener('click', function (e) {
    if (e.target === this) closeSettingsModal();
  });

  document.getElementById('setting-autofocus').addEventListener('change', function () {
    settings.autofocusSearch = this.checked;
    save(settings);
  });

  document.getElementById('setting-new-tab').addEventListener('change', function () {
    settings.openLinksNewTab = this.checked;
    save(settings);
    applyLinkTargets(settings.openLinksNewTab);
  });
})();

/* ── Three-dot card menu ────────────────────────────────── */
function closeAllMenus() {
  document.querySelectorAll('.card-menu.open').forEach(function (m) {
    m.classList.remove('open');
  });
}

document.addEventListener('click', function (e) {
  // Toggle menu on ⋯ button click.
  var btn = e.target.closest('.card-menu-btn');
  if (btn) {
    e.stopPropagation();
    var id   = btn.dataset.id;
    var menu = document.getElementById('menu-' + id);
    var wasOpen = menu.classList.contains('open');
    closeAllMenus();
    if (!wasOpen) menu.classList.add('open');
    return;
  }

  // Open edit modal when clicking the Edit menu item.
  var editBtn = e.target.closest('[data-action="edit"]');
  if (editBtn) {
    var d = editBtn.dataset;
    closeAllMenus();
    openEditModal(d.id, d.name, d.href, d.img);
    return;
  }

  // Dismiss all menus when clicking anywhere else outside a menu.
  if (!e.target.closest('.card-menu')) {
    closeAllMenus();
  }
});
