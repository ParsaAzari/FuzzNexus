<script lang="ts">
  import Icon from './Icon.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';

  interface Props {
    onOpenWordlists?: () => void;
  }

  let { onOpenWordlists }: Props = $props();

  let isExpanded = $state<boolean>(false);

  let alert = $derived(scanStore.advisoryAlert);

  let isWordlistTrigger = $derived(alert?.verdict === 'WORDLIST_WAF_TRIGGER');
  let isIPBan = $derived(alert?.verdict === 'IP_BAN');
  let isRateLimit = $derived(alert?.verdict === 'RATE_LIMIT');

  let badgeLabel = $derived.by(() => {
    if (!alert) return '';
    if (isWordlistTrigger) {
      return `IP HEALTHY (${alert.canary.root_status || 200} OK) · WAF PAYLOAD TRIGGER`;
    }
    if (isIPBan) return 'IP BLOCK CONFIRMED';
    if (isRateLimit) return 'RATE LIMIT (429)';
    return 'ENDPOINT PROTECTED';
  });

  async function handleSanitizeAndRescan() {
    try {
      const dto = await scanStore.sanitizeActiveWordlist(true);
      toastStore.success(
        `Created clean sandbox copy (${dto.line_count.toLocaleString()} safe lines) and restarted scan`
      );
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      toastStore.error(msg);
    }
  }
</script>

{#if alert}
  <section
    class="advisory-banner select-none"
    class:is-wordlist={isWordlistTrigger}
    class:is-ban={isIPBan || isRateLimit}
    class:is-expanded={isExpanded}
    role="status"
    aria-live="polite"
  >
    <!-- Main Compact Bar (Single-Line, Height ~34px) -->
    <div class="advisory-main-row">
      <div class="advisory-left">
        <div class="advisory-icon-wrap" class:is-wordlist={isWordlistTrigger} class:is-ban={isIPBan || isRateLimit}>
          <Icon name={isWordlistTrigger ? 'alert-triangle' : isIPBan ? 'shield-alert' : 'info'} size={14} />
        </div>

        <span class="advisory-pill font-mono" class:is-wordlist={isWordlistTrigger} class:is-ban={isIPBan || isRateLimit}>
          {badgeLabel}
        </span>

        <span class="advisory-title">{alert.title}</span>

        <button
          type="button"
          class="btn-toggle-details font-mono"
          onclick={() => (isExpanded = !isExpanded)}
          title="Toggle diagnostic telemetry and evidence chips"
        >
          <span>{isExpanded ? 'Hide Details' : 'Details'}</span>
          <Icon name={isExpanded ? 'chevron-up' : 'chevron-down'} size={11} />
        </button>
      </div>

      <div class="advisory-actions">
        {#if isWordlistTrigger}
          <button
            type="button"
            class="adv-btn adv-btn-primary"
            disabled={scanStore.isSanitizingWordlist}
            onclick={handleSanitizeAndRescan}
            title="Create a non-destructive sandbox wordlist removing traversal/bypass patterns (.//, .;/, ../) and rescan"
          >
            <Icon name="zap" size={11} />
            <span>{scanStore.isSanitizingWordlist ? 'Sanitizing...' : 'Sanitize & Rescan'}</span>
          </button>

          {#if onOpenWordlists}
            <button
              type="button"
              class="adv-btn adv-btn-secondary"
              onclick={onOpenWordlists}
              title="Switch to a clean directory or API wordlist"
            >
              <Icon name="layers" size={11} />
              <span>Switch Wordlist</span>
            </button>
          {/if}
        {/if}

        <button
          type="button"
          class="adv-btn adv-btn-ghost"
          onclick={() => scanStore.dismissAdvisory()}
          title="Dismiss advisory and continue current scan"
          aria-label="Dismiss advisory"
        >
          <Icon name="x" size={12} />
        </button>
      </div>
    </div>

    <!-- Collapsible Telemetry & Evidence Details Drawer -->
    {#if isExpanded}
      <div class="advisory-details-drawer">
        <p class="advisory-summary">{alert.summary}</p>

        <div class="advisory-evidence font-mono">
          {#if alert.canary?.root_status > 0}
            <span class="ev-chip ev-canary">
              Origin Canary: {alert.canary.root_status}
            </span>
          {/if}
          {#if alert.canary?.benign_path_status && alert.canary.benign_path_status > 0}
            <span class="ev-chip ev-canary">
              Benign Path: {alert.canary.benign_path_status}
            </span>
          {/if}
          {#if alert.evidence_header}
            <span class="ev-chip ev-header" title="Matched WAF response header">
              {alert.evidence_header}
            </span>
          {/if}
          {#if alert.fixed_body_length && alert.fixed_body_length > 0}
            <span class="ev-chip">
              Uniform body: {alert.fixed_body_length} B
            </span>
          {/if}
          {#if alert.sample_payloads && alert.sample_payloads.length > 0}
            <span class="ev-chip ev-payloads" title={alert.sample_payloads.join(', ')}>
              Triggers: {alert.sample_payloads.slice(0, 4).join(', ')}
            </span>
          {/if}
          <span class="ev-chip ev-confidence">
            Confidence: {alert.confidence}%
          </span>
        </div>
      </div>
    {/if}
  </section>
{/if}

<style>
  .advisory-banner {
    display: flex;
    flex-direction: column;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    border-left: 3px solid #f59e0b;
    font-size: var(--text-sm);
    animation: advSlideDown 0.18s ease-out;
    transition: all 0.15s ease;
  }

  .advisory-banner.is-wordlist {
    background-color: rgba(245, 158, 11, 0.06);
    border-left-color: #f59e0b;
  }

  .advisory-banner.is-ban {
    background-color: rgba(239, 68, 68, 0.06);
    border-left-color: #ef4444;
  }

  @keyframes advSlideDown {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .advisory-main-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    height: 34px;
    padding: 0 12px;
  }

  .advisory-left {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    flex: 1;
    overflow: hidden;
  }

  .advisory-icon-wrap {
    width: 20px;
    height: 20px;
    border-radius: var(--radius-xs);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background-color: rgba(245, 158, 11, 0.16);
    color: #d97706;
  }

  .advisory-icon-wrap.is-ban {
    background-color: rgba(239, 68, 68, 0.16);
    color: #ef4444;
  }

  .advisory-pill {
    font-size: 9.5px;
    font-weight: 700;
    letter-spacing: 0.03em;
    padding: 2px 6px;
    border-radius: var(--radius-xs);
    background-color: rgba(245, 158, 11, 0.16);
    color: #b45309;
    border: 1px solid rgba(245, 158, 11, 0.35);
    white-space: nowrap;
    flex-shrink: 0;
  }

  :global(:root[data-mode='dark']) .advisory-pill.is-wordlist {
    color: #fbbf24;
  }

  .advisory-pill.is-ban {
    background-color: rgba(239, 68, 68, 0.15);
    color: #dc2626;
    border: 1px solid rgba(239, 68, 68, 0.35);
  }

  :global(:root[data-mode='dark']) .advisory-pill.is-ban {
    color: #f87171;
  }

  .advisory-title {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-toggle-details {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 1px 6px;
    font-size: 10px;
    color: var(--text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.12s ease;
  }

  .btn-toggle-details:hover {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--accent);
  }

  .advisory-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .adv-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 22px;
    padding: 0 8px;
    font-size: 11px;
    font-weight: 500;
    border-radius: var(--radius-xs);
    cursor: pointer;
    border: 1px solid transparent;
    white-space: nowrap;
    transition: all 0.15s ease;
  }

  .adv-btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .adv-btn-primary {
    background-color: #d97706;
    color: #ffffff;
  }

  .adv-btn-primary:hover:not(:disabled) {
    background-color: #b45309;
  }

  .adv-btn-secondary {
    background-color: var(--surface);
    color: var(--text);
    border-color: var(--border);
  }

  .adv-btn-secondary:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .adv-btn-ghost {
    background-color: transparent;
    color: var(--text-secondary);
    padding: 0 5px;
  }

  .adv-btn-ghost:hover:not(:disabled) {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  /* Collapsible Details Drawer */
  .advisory-details-drawer {
    padding: 6px 12px 9px 36px;
    border-top: 1px dashed var(--border);
    display: flex;
    flex-direction: column;
    gap: 5px;
    animation: drawerOpen 0.15s ease;
  }

  @keyframes drawerOpen {
    from {
      opacity: 0;
      transform: translateY(-2px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .advisory-summary {
    margin: 0;
    font-size: 11px;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .advisory-evidence {
    display: flex;
    align-items: center;
    gap: 5px;
    flex-wrap: wrap;
  }

  .ev-chip {
    font-size: 9.5px;
    padding: 1px 5px;
    border-radius: var(--radius-xs);
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .ev-chip.ev-canary {
    color: #16a34a;
    border-color: rgba(34, 197, 94, 0.35);
  }

  :global(:root[data-mode='dark']) .ev-chip.ev-canary {
    color: #4ade80;
  }

  .ev-chip.ev-header {
    color: #d97706;
    border-color: rgba(245, 158, 11, 0.35);
  }

  :global(:root[data-mode='dark']) .ev-chip.ev-header {
    color: #fbbf24;
  }

  .ev-chip.ev-confidence {
    opacity: 0.8;
  }
</style>
