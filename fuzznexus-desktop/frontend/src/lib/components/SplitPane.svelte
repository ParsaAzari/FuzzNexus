<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    orientation?: 'vertical' | 'horizontal';
    initialSplit?: number; // percentage (e.g. 60)
    minSplit?: number;
    maxSplit?: number;
    first: Snippet;
    second: Snippet;
  }

  let {
    orientation = 'vertical',
    initialSplit = 60,
    minSplit = 20,
    maxSplit = 80,
    first,
    second
  }: Props = $props();

  let split = $state<number>(60);
  let isDragging = $state<boolean>(false);
  let containerRef = $state<HTMLDivElement | null>(null);

  $effect(() => {
    split = initialSplit;
  });

  function startDrag(e: MouseEvent) {
    e.preventDefault();
    isDragging = true;

    function onMouseMove(moveEvent: MouseEvent) {
      if (!isDragging || !containerRef) return;
      const rect = containerRef.getBoundingClientRect();
      let newSplit: number;

      if (orientation === 'vertical') {
        const offsetY = moveEvent.clientY - rect.top;
        newSplit = (offsetY / rect.height) * 100;
      } else {
        const offsetX = moveEvent.clientX - rect.left;
        newSplit = (offsetX / rect.width) * 100;
      }

      if (newSplit >= minSplit && newSplit <= maxSplit) {
        split = newSplit;
      }
    }

    function onMouseUp() {
      isDragging = false;
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    }

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  function resetSplit() {
    split = initialSplit;
  }
</script>

<div
  class="split-container {orientation}"
  bind:this={containerRef}
  class:dragging={isDragging}
>
  <div class="split-pane first" style="{orientation === 'vertical' ? `height: ${split}%;` : `width: ${split}%;`}">
    {@render first()}
  </div>

  <!-- Divider Handle -->
  <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
  <div
    class="split-divider {orientation}"
    role="separator"
    aria-label="Resize panels"
    aria-orientation={orientation}
    aria-valuenow={Math.round(split)}
    tabindex="0"
    onmousedown={startDrag}
    ondblclick={resetSplit}
    onkeydown={(e) => {
      if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        e.preventDefault();
        split = Math.max(minSplit, split - 2);
      } else if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
        e.preventDefault();
        split = Math.min(maxSplit, split + 2);
      }
    }}
  >
    <span class="divider-line"></span>
  </div>

  <div class="split-pane second" style="{orientation === 'vertical' ? `height: ${100 - split}%;` : `width: ${100 - split}%;`}">
    {@render second()}
  </div>
</div>

<style>
  .split-container {
    display: flex;
    width: 100%;
    height: 100%;
    overflow: hidden;
    position: relative;
  }

  .split-container.vertical {
    flex-direction: column;
  }

  .split-container.horizontal {
    flex-direction: row;
  }

  .split-container.dragging {
    cursor: row-resize;
    user-select: none;
  }

  .split-container.horizontal.dragging {
    cursor: col-resize;
  }

  .split-pane {
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .split-divider {
    background-color: var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
    transition: background 0.15s ease;
  }

  .split-divider.vertical {
    height: 6px;
    width: 100%;
    cursor: row-resize;
  }

  .split-divider.horizontal {
    width: 6px;
    height: 100%;
    cursor: col-resize;
  }

  .split-divider:hover,
  .split-container.dragging .split-divider {
    background-color: var(--accent);
  }

  .divider-line {
    border-radius: 2px;
  }

  .split-divider.vertical .divider-line {
    width: 32px;
    height: 2px;
    background-color: rgba(255, 255, 255, 0.4);
  }

  .split-divider.horizontal .divider-line {
    height: 32px;
    width: 2px;
    background-color: rgba(255, 255, 255, 0.4);
  }
</style>
