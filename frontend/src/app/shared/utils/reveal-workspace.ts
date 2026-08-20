/**
 * Reveals an inline workspace that has just been rendered below (or above) a
 * list. Two animation frames let Angular commit the signal-driven DOM before
 * measuring it. The first eligible form control receives keyboard focus.
 */
export function revealWorkspace(targetId: string, focusSelector = '[data-autofocus]'): void {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;

  const schedule = (callback: () => void): void => {
    if (typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(callback);
      return;
    }
    window.setTimeout(callback, 0);
  };

  schedule(() => {
    schedule(() => {
      const workspace = document.getElementById(targetId);
      if (!workspace) return;

      const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
      const stickyHeaderOffset = 144;
      const safeViewportBottom = window.innerHeight - 24;
      const bounds = workspace.getBoundingClientRect();
      const needsScroll = bounds.top < stickyHeaderOffset || bounds.bottom > safeViewportBottom;

      if (needsScroll) {
        workspace.scrollIntoView({
          behavior: reducedMotion ? 'auto' : 'smooth',
          block: 'start',
        });
      }

      const field = workspace.querySelector<HTMLElement>(focusSelector);
      field?.focus({ preventScroll: true });
    });
  });
}
