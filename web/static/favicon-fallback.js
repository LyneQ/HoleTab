// Handle favicon loading errors as early as possible
(function () {
  function useFallback(img) {
    if (img.src.indexOf('/static/favicon-fallback.svg') === -1) {
      img.src = '/static/favicon-fallback.svg';
      img.onerror = null;
      img.onload = null;
    }
  }

  // Handle network errors or 404s that browsers treat as errors
  document.addEventListener('error', function (e) {
    if (e.target.tagName === 'IMG' && e.target.classList.contains('link-favicon')) {
      useFallback(e.target);
    }
  }, true);

  // Handle Google's fallback globe (returns 404 with a 16x16 PNG)
  document.addEventListener('load', function (e) {
    var img = e.target;
    if (img.tagName === 'IMG' && img.classList.contains('link-favicon')) {
      // If we got a 16x16 image from Google's service (we always request 64)
      if (img.naturalWidth === 16 && (img.src.indexOf('gstatic.com') !== -1 || img.src.indexOf('google.com') !== -1)) {
        useFallback(img);
      }
    }
  }, true);
})();
