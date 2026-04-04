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

/* ── Add-link modal ─────────────────────────────────────── */
function openAddModal() {
  document.getElementById('add-modal').classList.add('open');
}

function closeAddModal() {
  document.getElementById('add-modal').classList.remove('open');
}

// Close modal after HTMX swaps the link grid (successful add).
document.addEventListener('htmx:afterSwap', function (e) {
  if (e.detail.target && e.detail.target.id === 'link-grid') {
    closeAddModal();
  }
});

// Close modal when clicking the backdrop itself.
document.getElementById('add-modal').addEventListener('click', function (e) {
  if (e.target === this) closeAddModal();
});

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

  // Dismiss all menus when clicking anywhere else outside a menu.
  if (!e.target.closest('.card-menu')) {
    closeAllMenus();
  }
});
