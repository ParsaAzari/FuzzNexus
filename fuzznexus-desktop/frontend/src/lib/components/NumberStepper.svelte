<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    label: string;
    value: number;
    min?: number;
    max?: number;
    step?: number;
    shiftStep?: number;
    disabled?: boolean;
    title?: string;
    ariaLabel?: string;
    width?: string;
    onchange?: (val: number) => void;
  }

  let {
    label,
    value = $bindable(0),
    min = 0,
    max = 999999,
    step = 1,
    shiftStep = 5,
    disabled = false,
    title = '',
    ariaLabel = '',
    width = 'auto',
    onchange
  }: Props = $props();

  function clamp(val: number): number {
    return Math.min(max, Math.max(min, val));
  }

  function handleIncrement(e: MouseEvent) {
    if (disabled) return;
    const current = Number(value) || 0;
    const delta = e.shiftKey ? shiftStep : step;
    const next = clamp(current + delta);
    value = next;
    onchange?.(next);
  }

  function handleDecrement(e: MouseEvent) {
    if (disabled) return;
    const current = Number(value) || 0;
    const delta = e.shiftKey ? shiftStep : step;
    const next = clamp(current - delta);
    value = next;
    onchange?.(next);
  }

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.value === '') return;
    const num = parseInt(target.value, 10);
    if (!isNaN(num)) {
      value = num;
      onchange?.(num);
    }
  }

  function handleBlur(e: Event) {
    const target = e.target as HTMLInputElement;
    let num = parseInt(target.value, 10);
    if (isNaN(num)) num = min;
    const clamped = clamp(num);
    value = clamped;
    target.value = String(clamped);
    onchange?.(clamped);
  }

  function handleWheel(e: WheelEvent) {
    if (disabled) return;
    e.preventDefault();
    const current = Number(value) || 0;
    const delta = e.deltaY < 0 ? (e.shiftKey ? shiftStep : step) : -(e.shiftKey ? shiftStep : step);
    const next = clamp(current + delta);
    value = next;
    onchange?.(next);
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (disabled) return;
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      const current = Number(value) || 0;
      const delta = e.shiftKey ? shiftStep : step;
      const next = clamp(current + delta);
      value = next;
      onchange?.(next);
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      const current = Number(value) || 0;
      const delta = e.shiftKey ? shiftStep : step;
      const next = clamp(current - delta);
      value = next;
      onchange?.(next);
    }
  }
</script>

<div
  class="stepper-group"
  class:is-disabled={disabled}
  style:width={width}
  {title}
>
  <span class="stepper-label">{label}</span>

  <input
    type="number"
    {min}
    {max}
    value={value}
    {disabled}
    oninput={handleInput}
    onblur={handleBlur}
    onwheel={handleWheel}
    onkeydown={handleKeyDown}
    class="stepper-input font-mono"
    aria-label={ariaLabel || label}
  />

  <div class="stepper-column">
    <button
      type="button"
      class="stepper-btn stepper-up"
      onclick={handleIncrement}
      disabled={disabled || value >= max}
      title="Increase value (Up Arrow)"
      aria-label={`Increase ${label}`}
    >
      <Icon name="chevron-up" size={9} />
    </button>
    <button
      type="button"
      class="stepper-btn stepper-down"
      onclick={handleDecrement}
      disabled={disabled || value <= min}
      title="Decrease value (Down Arrow)"
      aria-label={`Decrease ${label}`}
    >
      <Icon name="chevron-down" size={9} />
    </button>
  </div>
</div>

<style>
  .stepper-group {
    display: inline-flex;
    align-items: center;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    height: var(--control-h-base);
    padding-left: 8px;
    padding-right: 0;
    box-sizing: border-box;
    transition: border-color 0.15s ease;
    user-select: none;
    flex-shrink: 0;
  }

  .stepper-group:focus-within {
    border-color: var(--accent);
  }

  .stepper-group.is-disabled {
    opacity: 0.55;
  }

  .stepper-label {
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.03em;
    white-space: nowrap;
    margin-right: 4px;
    flex-shrink: 0;
  }

  .stepper-input {
    all: unset;
    flex: 1;
    font-family: var(--font-mono);
    font-size: var(--text-base);
    font-variant-numeric: tabular-nums;
    color: var(--text);
    text-align: right;
    padding: 0 7px;
    min-width: 0;
    box-sizing: border-box;
  }

  /* Completely eliminate browser native spin buttons */
  .stepper-input::-webkit-inner-spin-button,
  .stepper-input::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
    display: none;
  }

  .stepper-input {
    appearance: textfield;
    -moz-appearance: textfield;
    -webkit-appearance: none;
  }

  .stepper-column {
    display: flex;
    flex-direction: column;
    width: 20px;
    height: 100%;
    border-left: 1px solid var(--border);
    background-color: var(--surface);
    border-top-right-radius: calc(var(--radius-sm) - 1px);
    border-bottom-right-radius: calc(var(--radius-sm) - 1px);
    overflow: hidden;
    flex-shrink: 0;
  }

  .stepper-btn {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 50%;
    width: 100%;
    cursor: pointer;
    color: var(--text-secondary);
    background: transparent;
    transition: background-color 0.12s ease, color 0.12s ease;
    user-select: none;
    box-sizing: border-box;
  }

  .stepper-btn:hover:not(:disabled) {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  .stepper-btn:active:not(:disabled) {
    background-color: var(--accent);
    color: var(--accent-contrast, #ffffff);
  }

  .stepper-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .stepper-up {
    border-bottom: 1px solid var(--border);
  }
</style>
