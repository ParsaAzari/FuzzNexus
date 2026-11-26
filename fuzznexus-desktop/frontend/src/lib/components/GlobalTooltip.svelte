<script lang="ts">
  import { onMount } from 'svelte';

  interface TooltipState {
    text: string;
    visible: boolean;
    x: number;
    y: number;
    placement: 'top' | 'bottom';
  }

  let tooltip = $state<TooltipState>({
    text: '',
    visible: false,
    x: 0,
    y: 0,
    placement: 'top'
  });

  let tooltipEl = $state<HTMLDivElement | null>(null);
  let showTimeout: any = null;
  let currentTarget: HTMLElement | null = null;

  function findTooltipTarget(target: EventTarget | null): HTMLElement | null {
    let el = target as HTMLElement | null;
    while (el && el !== document.body) {
      if (el.hasAttribute('data-tooltip')) {
        const text = el.getAttribute('data-tooltip')?.trim();
        if (text) return el;
      }
      el = el.parentElement;
    }
    return null;
  }

  function updatePosition(el: HTMLElement) {
    const rect = el.getBoundingClientRect();
    const preferredPos = el.getAttribute('data-tooltip-pos') || 'top';
    const padding = 10;
    const gap = 7;

    // Viewport dimensions
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    // Tooltip dimensions estimation or measured
    const ttWidth = tooltipEl ? tooltipEl.offsetWidth : 200;
    const ttHeight = tooltipEl ? tooltipEl.offsetHeight : 28;

    // Determine vertical placement: top or bottom
    let placement: 'top' | 'bottom' = 'top';
    if (preferredPos === 'down') {
      placement = 'bottom';
    }

    // Edge collision detection for top/bottom
    if (placement === 'top' && rect.top - ttHeight - gap < padding) {
      // Not enough space above, flip to bottom
      placement = 'bottom';
    } else if (placement === 'bottom' && rect.bottom + ttHeight + gap > vh - padding) {
      // Not enough space below, flip to top
      placement = 'top';
    }

    let y: number;
    if (placement === 'top') {
      y = rect.top - gap;
    } else {
      y = rect.bottom + gap;
    }

    // Determine horizontal center and clamp inside window bounds
    const elementCenter = rect.left + rect.width / 2;
    let x = elementCenter;

    // Clamping to ensure tooltip never overflows viewport edges
    const halfWidth = ttWidth / 2;
    if (x - halfWidth < padding) {
      x = padding + halfWidth;
    } else if (x + halfWidth > vw - padding) {
      x = vw - padding - halfWidth;
    }

    tooltip.placement = placement;
    tooltip.x = Math.round(x);
    tooltip.y = Math.round(y);
  }

  function handleMouseOver(e: MouseEvent) {
    const target = findTooltipTarget(e.target);
    if (!target) {
      hide();
      return;
    }

    if (target === currentTarget && tooltip.visible) {
      return;
    }

    currentTarget = target;
    const text = target.getAttribute('data-tooltip') || '';
    if (!text) {
      hide();
      return;
    }

    clearTimeout(showTimeout);
    // Instant switch if another tooltip was already visible, else quick 140ms grace delay
    const delay = tooltip.visible ? 0 : 140;

    showTimeout = setTimeout(() => {
      if (!currentTarget) return;
      tooltip.text = text;
      tooltip.visible = true;
      // Allow DOM to render then adjust precise clamped position
      requestAnimationFrame(() => {
        if (currentTarget) {
          updatePosition(currentTarget);
        }
      });
    }, delay);
  }

  function handleMouseOut(e: MouseEvent) {
    const related = e.relatedTarget as HTMLElement | null;
    const target = findTooltipTarget(related);
    if (target && target === currentTarget) {
      return; // Still within same tooltip target
    }
    hide();
  }

  function hide() {
    clearTimeout(showTimeout);
    currentTarget = null;
    tooltip.visible = false;
  }

  function handleScrollOrResize() {
    if (tooltip.visible && currentTarget) {
      updatePosition(currentTarget);
    }
  }

  onMount(() => {
    window.addEventListener('mouseover', handleMouseOver, { passive: true });
    window.addEventListener('mouseout', handleMouseOut, { passive: true });
    window.addEventListener('scroll', handleScrollOrResize, { passive: true, capture: true });
    window.addEventListener('resize', handleScrollOrResize, { passive: true });
    window.addEventListener('blur', hide, { passive: true });

    return () => {
      clearTimeout(showTimeout);
      window.removeEventListener('mouseover', handleMouseOver);
      window.removeEventListener('mouseout', handleMouseOut);
      window.removeEventListener('scroll', handleScrollOrResize, { capture: true });
      window.removeEventListener('resize', handleScrollOrResize);
      window.removeEventListener('blur', hide);
    };
  });
</script>

{#if tooltip.visible && tooltip.text}
  <div
    bind:this={tooltipEl}
    class="portal-floating-tooltip"
    class:placement-top={tooltip.placement === 'top'}
    class:placement-bottom={tooltip.placement === 'bottom'}
    style="left: {tooltip.x}px; top: {tooltip.y}px;"
    role="tooltip"
    aria-hidden="true"
  >
    {tooltip.text}
  </div>
{/if}

<style>
  .portal-floating-tooltip {
    position: fixed;
    z-index: 9999999;
    pointer-events: none;
    max-width: 320px;
    padding: 5px 9px;
    background-color: var(--surface, #1e1e24);
    color: var(--text, #f0f0f5);
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: var(--radius-sm, 6px);
    font-family: var(--font-sans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif);
    font-size: 11px;
    font-weight: 500;
    line-height: 1.35;
    letter-spacing: -0.01em;
    white-space: normal;
    word-break: break-word;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.28), 0 1px 3px rgba(0, 0, 0, 0.16);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    will-change: transform, opacity;
    animation: tooltip-fade-in 0.15s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  .portal-floating-tooltip.placement-top {
    transform: translate(-50%, -100%);
  }

  .portal-floating-tooltip.placement-bottom {
    transform: translate(-50%, 0);
  }

  @keyframes tooltip-fade-in {
    from {
      opacity: 0;
      filter: blur(2px);
    }
    to {
      opacity: 1;
      filter: blur(0);
    }
  }
</style>
