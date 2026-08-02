document.addEventListener('DOMContentLoaded', function () {
  var toggle = document.getElementById('nav-toggle');
  var mobile = document.getElementById('mobile-nav');
  var iconMenu = document.getElementById('icon-menu');
  var iconClose = document.getElementById('icon-close');
  if (toggle && mobile) {
    toggle.addEventListener('click', function () {
      var open = mobile.classList.contains('opacity-100');
      if (open) {
        mobile.classList.remove('opacity-100', 'pointer-events-auto');
        mobile.classList.add('opacity-0', 'pointer-events-none');
        if (iconMenu) iconMenu.classList.remove('hidden');
        if (iconClose) iconClose.classList.add('hidden');
      } else {
        mobile.classList.add('opacity-100', 'pointer-events-auto');
        mobile.classList.remove('opacity-0', 'pointer-events-none');
        if (iconMenu) iconMenu.classList.add('hidden');
        if (iconClose) iconClose.classList.remove('hidden');
      }
    });
    mobile.querySelectorAll('a').forEach(function (a) {
      a.addEventListener('click', function () {
        mobile.classList.remove('opacity-100', 'pointer-events-auto');
        mobile.classList.add('opacity-0', 'pointer-events-none');
        if (iconMenu) iconMenu.classList.remove('hidden');
        if (iconClose) iconClose.classList.add('hidden');
      });
    });
  }

  // Global starfield background (parity with React Index.tsx; z-[1] so visible above body paint)
  var stars = document.getElementById('cyber-stars');
  if (stars && !stars.dataset.ready) {
    stars.dataset.ready = '1';
    for (var i = 0; i < 100; i++) {
      var star = document.createElement('div');
      star.className = 'cyber-star';
      var size = Math.random() * 3 + 1;
      var opacity = Math.random() * 0.5 + 0.25;
      var duration = Math.random() * 3 + 2;
      star.style.left = Math.random() * 100 + '%';
      star.style.top = Math.random() * 100 + '%';
      star.style.width = size + 'px';
      star.style.height = size + 'px';
      star.style.setProperty('--star-opacity', String(opacity));
      star.style.setProperty('--star-duration', duration + 's');
      stars.appendChild(star);
    }
  }

  // Project filter active state
  var filters = document.getElementById('project-filters');
  if (filters) {
    filters.addEventListener('click', function (e) {
      var btn = e.target.closest('.filter-btn');
      if (!btn) return;
      filters.querySelectorAll('.filter-btn').forEach(function (b) {
        b.classList.remove('is-active', 'bg-gradient-to-r', 'from-cyber-purple/20', 'to-cyber-cyan/20', 'text-cyber-cyan', 'border', 'border-cyber-cyan/30', 'shadow-[0_0_10px_rgba(0,255,255,0.2)]');
        b.classList.add('text-cyber-lavender/70', 'hover:text-cyber-lavender');
      });
      btn.classList.add('is-active', 'bg-gradient-to-r', 'from-cyber-purple/20', 'to-cyber-cyan/20', 'text-cyber-cyan', 'border', 'border-cyber-cyan/30', 'shadow-[0_0_10px_rgba(0,255,255,0.2)]');
      btn.classList.remove('text-cyber-lavender/70', 'hover:text-cyber-lavender');
    });
  }

  var copyBtn = document.getElementById('copy-link');
  if (copyBtn) {
    copyBtn.addEventListener('click', function () {
      var url = copyBtn.getAttribute('data-url');
      if (navigator.clipboard && url) {
        navigator.clipboard.writeText(url).then(function () {
          copyBtn.textContent = 'Copied!';
          setTimeout(function () { copyBtn.textContent = 'Copy link'; }, 1500);
        });
      }
    });
  }

  document.body.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeProjectModal();
  });
});

function closeProjectModal() {
  var root = document.getElementById('project-modal-root');
  if (root) root.innerHTML = '';
}
