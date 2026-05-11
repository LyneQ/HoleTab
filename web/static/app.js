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

/* ── Latency ping ───────────────────────────────────────── */
(function () {
  var TARGETS = {
    cloudflare: 'https://www.cloudflare.com/cdn-cgi/trace',
    google:     'https://connectivitycheck.gstatic.com/generate_204'
  };

  var badge  = document.getElementById('latency-badge');
  var handle = null;
  var active = false;

  function colorClass(ms) {
    return ms < 80 ? 'latency-good' : ms < 200 ? 'latency-mid' : 'latency-bad';
  }

  function ping(target) {
    var url = (TARGETS[target] || TARGETS.cloudflare) + '?_=' + Date.now();
    var t0  = performance.now();
    fetch(url, { method: 'GET', cache: 'no-store', mode: 'no-cors' })
      .then(function () {
        var ms = Math.round(performance.now() - t0);
        badge.textContent = ms + ' ms';
        badge.className   = 'latency-badge ' + colorClass(ms);
      })
      .catch(function () {
        badge.textContent = '-- ms';
        badge.className   = 'latency-badge latency-bad';
      });
  }

  function startLoop(target) {
    if (handle) return;
    ping(target);
    handle = setInterval(function () { ping(target); }, 5000);
  }

  function stopLoop() {
    clearInterval(handle);
    handle = null;
  }

  window.latencyControl = {
    _target: 'cloudflare',

    enable: function (target) {
      this._target = target || 'cloudflare';
      badge.style.display = 'block';
      active = true;
      if (document.visibilityState !== 'hidden') startLoop(this._target);
    },

    disable: function () {
      active = false;
      stopLoop();
      badge.style.display = 'none';
    },

    setTarget: function (target) {
      this._target = target;
      if (active) { stopLoop(); startLoop(target); }
    }
  };

  document.addEventListener('visibilitychange', function () {
    if (!active) return;
    if (document.visibilityState === 'hidden') {
      stopLoop();
    } else {
      startLoop(window.latencyControl._target);
    }
  });
})();

/* ── Settings ───────────────────────────────────────────── */
(function () {
  var KEY = 'holetab-settings';
  var DEFAULTS = { autofocusSearch: false, openLinksNewTab: true, showLatency: false, latencyTarget: 'cloudflare' };

  function load() {
    try {
      var s = localStorage.getItem(KEY);
      return s ? Object.assign({}, DEFAULTS, JSON.parse(s)) : Object.assign({}, DEFAULTS);
    } catch (e) { return Object.assign({}, DEFAULTS); }
  }

  function save(s) { localStorage.setItem(KEY, JSON.stringify(s)); }

  function applyLatencyTargetRow(show) {
    var row = document.getElementById('latency-target-row');
    if (row) row.style.display = show ? '' : 'none';
  }

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
  applyLatencyTargetRow(settings.showLatency);
  if (settings.showLatency) latencyControl.enable(settings.latencyTarget);

  window.openSettingsModal = function () {
    document.getElementById('setting-autofocus').checked        = settings.autofocusSearch;
    document.getElementById('setting-new-tab').checked          = settings.openLinksNewTab;
    document.getElementById('setting-latency').checked          = settings.showLatency;
    document.getElementById('setting-latency-target').value     = settings.latencyTarget;
    applyLatencyTargetRow(settings.showLatency);
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

  document.getElementById('setting-latency').addEventListener('change', function () {
    settings.showLatency = this.checked;
    save(settings);
    applyLatencyTargetRow(settings.showLatency);
    if (settings.showLatency) {
      latencyControl.enable(settings.latencyTarget);
    } else {
      latencyControl.disable();
    }
  });

  document.getElementById('setting-latency-target').addEventListener('change', function () {
    settings.latencyTarget = this.value;
    save(settings);
    latencyControl.setTarget(settings.latencyTarget);
  });
})();

/* ── Drag and Drop ──────────────────────────────────────── */
var sortableInstance = null;
var organizeModeActive = false; // Persistent state for the mode
var selectedIds = []; // Track selected IDs for multi-reorder

function disableDragAndDrop() {
  organizeModeActive = false;
  var grid = document.getElementById('link-grid');
  if (grid) {
    grid.classList.remove('organize-mode');
    grid.querySelectorAll('.link-card.selected').forEach(function (el) {
      el.classList.remove('selected');
    });
  }
  selectedIds = [];
  if (sortableInstance) {
    sortableInstance.destroy();
    sortableInstance = null;
  }
}

function enableDragAndDrop() {
  var grid = document.getElementById('link-grid');
  if (!grid) return;

  organizeModeActive = true;
  grid.classList.add('organize-mode');

  // Re-apply selected class if IDs were already selected
  selectedIds.forEach(function (id) {
    var el = document.getElementById('link-' + id);
    if (el) el.classList.add('selected');
  });

  // Cleanup stale or existing instance
  if (sortableInstance) {
    if (sortableInstance.el === grid) return;
    sortableInstance.destroy();
  }

  sortableInstance = new Sortable(grid, {
    animation: 150,
    ghostClass: 'sortable-ghost',
    draggable: '.link-card:not(.add-tile):not(.done-tile)',
    filter: '.card-menu-btn, .card-menu', // don't drag when clicking menu
    onEnd: function (evt) {
      var allCards = Array.from(grid.querySelectorAll('.link-card:not(.add-tile):not(.done-tile)'));
      var ids = allCards.map(function (el) { return el.id.replace('link-', ''); });

      var draggedId = evt.item.id.replace('link-', '');

      if (selectedIds.includes(draggedId)) {
        // Dragged a selected item, move all selected items together
        var nonSelectedIds = ids.filter(function (id) { return !selectedIds.includes(id); });
        var landingIdx = ids.indexOf(draggedId);

        // Find how many selected items are before the landingIdx in the current DOM order
        var selectedBefore = ids.slice(0, landingIdx).filter(function (id) {
          return selectedIds.includes(id);
        }).length;
        var insertAt = landingIdx - selectedBefore;

        // Construct new list: group all selected items at the insertion point
        ids = nonSelectedIds.slice(0, insertAt)
          .concat(selectedIds)
          .concat(nonSelectedIds.slice(insertAt));
      }

      // Use HTMX to send the new order and update the grid
      htmx.ajax('PUT', '/links/reorder', {
        target: '#link-grid',
        swap: 'outerHTML', // FIX: prevent nesting and fix the "cannot move other element" bug
        values: { ids: ids.join(',') }
      });
    }
  });
}

window.disableDragAndDrop = disableDragAndDrop;
window.enableDragAndDrop  = enableDragAndDrop;

// Consolidated HTMX loading and swap handling
htmx.onLoad(function (content) {
  var grid = content.id === 'link-grid' ? content : content.querySelector('#link-grid');
  if (grid) {
    // Close modals on successful grid update
    closeAddModal();
    closeEditModal();

    // Re-enable organize mode if it was active
    if (organizeModeActive) {
      // Prune selectedIds that are no longer in the DOM
      selectedIds = selectedIds.filter(function (id) {
        return document.getElementById('link-' + id);
      });
      enableDragAndDrop();
    }
  }
});

/* ── Three-dot card menu ────────────────────────────────── */
function closeAllMenus() {
  document.querySelectorAll('.card-menu.open').forEach(function (m) {
    m.classList.remove('open');
  });
  document.querySelectorAll('.link-card.menu-open').forEach(function (c) {
    c.classList.remove('menu-open');
  });
}

document.addEventListener('click', function (e) {
  var isOrganizeMode = organizeModeActive;

  // Toggle menu on ⋯ button click.
  var btn = e.target.closest('.card-menu-btn');
  if (btn) {
    e.stopPropagation();
    var id   = btn.dataset.id;
    var menu = document.getElementById('menu-' + id);
    var wasOpen = menu.classList.contains('open');
    closeAllMenus();
    if (!wasOpen) {
      menu.classList.add('open');
      btn.closest('.link-card').classList.add('menu-open');
    }
    return;
  }

  // Handle selection in organize mode.
  var card = e.target.closest('.link-card:not(.add-tile):not(.done-tile)');
  if (isOrganizeMode && card && !e.target.closest('.card-menu-btn') && !e.target.closest('.card-menu')) {
    e.preventDefault();
    e.stopPropagation();
    var cardId = card.id.replace('link-', '');
    var idx = selectedIds.indexOf(cardId);
    if (idx > -1) {
      selectedIds.splice(idx, 1);
      card.classList.remove('selected');
    } else {
      selectedIds.push(cardId);
      card.classList.add('selected');
    }
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

  // Handle Organize action.
  var organizeBtn = e.target.closest('[data-action="organize"]');
  if (organizeBtn) {
    closeAllMenus();
    enableDragAndDrop();
    return;
  }

  // Dismiss all menus when clicking anywhere else outside a menu.
  if (!e.target.closest('.card-menu')) {
    closeAllMenus();
  }
});
