// KYBa KCP UI glue for the Gitea fork overlay.
// The initial implementation is server-rendered. This file is intentionally small
// and only adds progressive enhancement hooks used by future Gitea templates.
export function initKCPCapsules() {
  document.querySelectorAll('[data-kcp-confirm]').forEach((element) => {
    element.addEventListener('click', (event) => {
      const message = element.getAttribute('data-kcp-confirm') || 'Continue?';
      if (!window.confirm(message)) event.preventDefault();
    });
  });
}

document.addEventListener('DOMContentLoaded', initKCPCapsules);
