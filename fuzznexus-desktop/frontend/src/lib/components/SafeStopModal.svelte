<script lang="ts">
  import Icon from './Icon.svelte';
  import { scanStore } from '../stores/scan.svelte';

  let prompt = $derived(scanStore.safeStopPrompt);
  let isTesting = $derived(scanStore.isTestingConnection);
  let testResult = $derived(scanStore.testConnectionResult);

  let isRawIP = $state(false);
  let copyFeedback = $state(false);

  let isVerified = $derived(testResult?.success === true);

  let ipChanged = $derived.by(() => {
    if (testResult?.new_raw_ip && prompt?.raw_ip) {
      return testResult.new_raw_ip !== prompt.raw_ip;
    }
    if (testResult?.new_ip_hash && prompt?.ip_hash) {
      return testResult.new_ip_hash !== prompt.ip_hash;
    }
    return false;
  });

  let activeIPHash = $derived(
    testResult?.new_ip_hash
      ? testResult.new_ip_hash
      : (prompt?.ip_hash && prompt.ip_hash !== 'unknown' ? prompt.ip_hash : '')
  );

  let activeRawIP = $derived(
    testResult?.new_raw_ip
      ? testResult.new_raw_ip
      : (prompt?.raw_ip || '')
  );

  let displayedIP = $derived.by(() => {
    if (isRawIP) {
      if (activeRawIP) {
        return activeRawIP;
      }
      if (activeIPHash) {
        return activeIPHash;
      }
      return '127.0.0.1';
    }
    if (activeIPHash) {
      return activeIPHash;
    }
    if (activeRawIP) {
      return activeRawIP;
    }
    return isVerified ? 'Verified Clean Route' : '12ca17b4';
  });

  function toggleIPMode() {
    isRawIP = !isRawIP;
  }

  function handleCopy() {
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(displayedIP);
      copyFeedback = true;
      setTimeout(() => {
        copyFeedback = false;
      }, 1800);
    }
  }

  function handleTest() {
    scanStore.testConnection();
  }

  function handleResume() {
    scanStore.resumeSafeStop();
  }

  function handleResumeAnyway() {
    scanStore.resumeAnyway();
  }

  function handleCancel() {
    scanStore.cancelSafeStop();
  }
</script>

{#if prompt}
  <div class="modal-backdrop" role="presentation">
    <div
      class="safe-stop-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="safe-stop-title"
    >
      <!-- Top Accent Line -->
      <div class="top-accent-bar" class:verified={isVerified}></div>

      <div class="dialog-content">
        <!-- Header: Perfectly Aligned with Native Window Dragging -->
        <div class="dialog-header" style="--wails-draggable: drag;">
          <div class="header-icon-box" class:verified={isVerified} style="--wails-draggable: drag;">
            {#if isVerified}
              <Icon name="check" size={18} strokeWidth={2.5} color="#4ade80" />
            {:else}
              <Icon name="alert-triangle" size={18} strokeWidth={2} color="#fbbf24" />
            {/if}
          </div>

          <div class="header-text" style="--wails-draggable: drag;">
            <div class="title-row" style="--wails-draggable: drag;">
              <h2 id="safe-stop-title" class="dialog-title">
                {#if isVerified}
                  New Route Verified
                {:else if ipChanged}
                  New IP Detected (Still Blocked)
                {:else}
                  Target Connection Blocked
                {/if}
              </h2>
              <span class="status-pill" class:verified={isVerified}>
                <span class="pulse-dot" class:verified={isVerified}></span>
                <span>{isVerified ? 'READY' : 'PAUSED'}</span>
              </span>
            </div>
            <p class="dialog-subtitle">
              {#if isVerified}
                Ready to resume scan seamlessly from paused index
              {:else if ipChanged}
                IP changed, but target is still rejecting traffic (403 Forbidden)
              {:else}
                WAF 403 Forbidden detected &bull; 0 requests leaking
              {/if}
            </p>
          </div>
        </div>

        <!-- Description Note: Explicitly FuzzNexus -->
        <div class="description-card">
          <p class="description-text">
            The remote server rejected traffic. FuzzNexus automatically froze the scan to prevent a permanent IP blacklist and preserve scan state.
          </p>
          {#if prompt.canary_summary}
            <p class="canary-proof font-mono">
              {prompt.canary_summary}
            </p>
          {/if}
        </div>

        <!-- Morphing IP Route Card -->
        <div
          class="ip-route-card"
          class:testing={isTesting}
          class:verified={isVerified}
        >
          <div class="card-inner">
            <!-- Card Upper Row: Labels & Equal Size Status Badges -->
            <div class="card-header-row">
              <div class="ip-label-group">
                <span class="ip-type-label" class:verified={isVerified}>
                  {#if isVerified}
                    NEW ACTIVE IP ADDRESS
                  {:else if ipChanged}
                    NEW DETECTED IP (STILL BLOCKED)
                  {:else}
                    BLOCKED IP ADDRESS
                  {/if}
                </span>
                <span class="tag-badge">
                  {isRawIP ? 'RAW IP' : 'SHA-256 (8c)'}
                </span>
              </div>

              <!-- Status Tag: Exactly matching size with tag-badge -->
              <div
                class="status-tag"
                class:testing={isTesting}
                class:verified={isVerified}
                class:error={testResult && !testResult.success}
              >
                <span
                  class="tag-dot"
                  class:testing={isTesting}
                  class:verified={isVerified}
                  class:error={testResult && !testResult.success}
                ></span>
                <span>
                  {#if isTesting}
                    Probing Target...
                  {:else if isVerified}
                    200 OK (Clean Route)
                  {:else if testResult && !testResult.success}
                    {#if ipChanged}
                      Target Still Blocked (403)
                    {:else}
                      IP Unchanged (Blocked)
                    {/if}
                  {:else}
                    403 Forbidden (Blocked)
                  {/if}
                </span>
              </div>
            </div>

            <!-- Card Middle Row: Monospace IP Display & Controls -->
            <div class="ip-value-row">
              <div class="ip-value-group">
                <span class="globe-icon" class:verified={isVerified}>
                  <Icon name="globe" size={16} color={isVerified ? '#22c55e' : '#fbbf24'} />
                </span>
                <span class="ip-code font-mono" class:is-raw={isRawIP}>
                  {displayedIP}
                </span>
              </div>

              <div class="ip-actions-group">
                <button
                  type="button"
                  class="action-icon-btn"
                  onclick={toggleIPMode}
                  title={isRawIP ? 'Switch to IP Hash' : 'Reveal Raw IP Address'}
                >
                  <Icon name={isRawIP ? 'eye-off' : 'eye'} size={14} />
                </button>

                <button
                  type="button"
                  class="action-icon-btn"
                  onclick={handleCopy}
                  title="Copy IP to clipboard"
                >
                  {#if copyFeedback}
                    <Icon name="check" size={14} color="#34d399" />
                  {:else}
                    <Icon name="copy" size={14} />
                  {/if}
                </button>
              </div>
            </div>

            <!-- Card Bottom Hint Banner -->
            <div class="hint-banner">
              <span class="hint-icon">
                {#if isVerified}
                  <Icon name="check" size={13} color="#22c55e" strokeWidth={2.5} />
                {:else if isTesting}
                  <Icon name="refresh" size={13} class="spin-icon" color="#818cf8" />
                {:else if testResult && !testResult.success}
                  <Icon name="alert-circle" size={13} color="#ef4444" />
                {:else}
                  <Icon name="info" size={13} color="#fbbf24" />
                {/if}
              </span>
              <span class="hint-text" class:verified={isVerified}>
                {#if copyFeedback}
                  Copied to clipboard.
                {:else if isTesting}
                  Testing new IP address and target reachability...
                {:else if isVerified}
                  Target is reachable from new IP. You can safely resume scan.
                {:else if testResult && !testResult.success}
                  {#if ipChanged}
                    New IP ({displayedIP}) detected, but target server is still returning 403 Forbidden. Try another VPN location or click Resume Anyway.
                  {:else}
                    IP address has not changed (still {displayedIP}). Please ensure your VPN is active.
                  {/if}
                {:else}
                  Switch your VPN or proxy IP, then click <strong>Test Connection</strong> to verify.
                {/if}
              </span>
            </div>
          </div>
        </div>

        <!-- Dialog Footer Action Buttons -->
        <div class="dialog-footer" style="--wails-draggable: no-drag;">
          <div class="footer-left">
            <button
              type="button"
              class="btn btn-cancel"
              onclick={handleCancel}
              title="Halt scan execution and persist encrypted resume state to disk"
            >
              <span>Cancel Scan</span>
            </button>
          </div>

          <div class="footer-right">
            <button
              type="button"
              class="btn btn-secondary"
              onclick={handleResumeAnyway}
              title="Force resume scan without changing IP"
            >
              <span>Resume Anyway</span>
            </button>

            <button
              type="button"
              class="btn btn-secondary-action"
              onclick={handleTest}
              disabled={isTesting}
              title="Probe target endpoint to verify new IP connectivity"
            >
              {#if isTesting}
                <span class="spin-icon">
                  <Icon name="refresh" size={13} />
                </span>
                <span>Testing...</span>
              {:else}
                <Icon name="refresh" size={13} />
                <span>{isVerified ? 'Re-test' : 'Test Connection'}</span>
              {/if}
            </button>

            <button
              type="button"
              class="btn btn-resume-cta"
              class:verified={isVerified}
              onclick={handleResume}
              disabled={!isVerified && !prompt}
              title="Resume scanning from last paused index"
            >
              <span>Resume Scan</span>
              <Icon name="chevron-right" size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background-color: rgba(5, 8, 15, 0.82);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    animation: fadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .safe-stop-dialog {
    width: 100%;
    max-width: 550px;
    background-color: #0f1623;
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow: 0 20px 45px -10px rgba(0, 0, 0, 0.7), 0 0 1px 1px rgba(255, 255, 255, 0.05);
    border-radius: 14px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    animation: scaleIn 0.22s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes scaleIn {
    from {
      transform: scale(0.96);
      opacity: 0;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }

  .top-accent-bar {
    height: 2px;
    width: 100%;
    background-color: rgba(245, 158, 11, 0.7);
    transition: background-color 0.25s ease;
  }

  .top-accent-bar.verified {
    background-color: rgba(34, 197, 94, 0.7);
  }

  .dialog-content {
    padding: 22px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Header */
  .dialog-header {
    display: flex;
    align-items: flex-start;
    gap: 14px;
  }

  .header-icon-box {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background-color: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.22);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: #fbbf24;
    transition: all 0.25s ease;
  }

  .header-icon-box.verified {
    background-color: rgba(34, 197, 94, 0.1);
    border-color: rgba(34, 197, 94, 0.22);
    color: #4ade80;
  }

  .header-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .dialog-title {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: #f8fafc;
    letter-spacing: -0.01em;
    line-height: 1.25;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 19px;
    padding: 0 7px;
    border-radius: 9999px;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.04em;
    background-color: rgba(245, 158, 11, 0.12);
    color: #fbbf24;
    border: 1px solid rgba(245, 158, 11, 0.25);
    line-height: 1;
    transition: all 0.25s ease;
  }

  .status-pill.verified {
    background-color: rgba(34, 197, 94, 0.12);
    color: #4ade80;
    border-color: rgba(34, 197, 94, 0.25);
  }

  .pulse-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background-color: #fbbf24;
    animation: pulse 1.8s infinite ease-in-out;
  }

  .pulse-dot.verified {
    background-color: #4ade80;
    animation: none;
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.4;
      transform: scale(0.8);
    }
  }

  .dialog-subtitle {
    margin: 0;
    font-size: 12px;
    color: #94a3b8;
    line-height: 1.35;
  }

  /* Description Card */
  .description-card {
    background-color: rgba(15, 23, 42, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 9px;
    padding: 10px 13px;
  }

  .description-text {
    margin: 0;
    font-size: 12px;
    color: #cbd5e1;
    line-height: 1.45;
  }

  .canary-proof {
    margin: 6px 0 0;
    padding-top: 6px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    font-size: 10.5px;
    color: #fbbf24;
    line-height: 1.35;
  }

  /* IP Route Card */
  .ip-route-card {
    background-color: #131b28;
    border: 1px solid rgba(245, 158, 11, 0.25);
    border-radius: 11px;
    padding: 13px 15px;
    position: relative;
    overflow: hidden;
    transition: all 0.25s ease;
  }

  .ip-route-card.testing {
    border-color: rgba(99, 102, 241, 0.25);
    transform: scale(0.98);
    opacity: 0.85;
  }

  .ip-route-card.verified {
    background-color: #131b28;
    border-color: rgba(34, 197, 94, 0.25);
  }

  .card-inner {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 11px;
  }

  /* Card Header Row: Equal dimensions for both badges */
  .card-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .ip-label-group {
    display: flex;
    align-items: center;
    gap: 7px;
  }

  .ip-type-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: #94a3b8;
    line-height: 1;
  }

  .ip-type-label.verified {
    color: #94a3b8;
  }

  /* Shared Exact Metric for Both Badges */
  .tag-badge,
  .status-tag {
    height: 20px;
    padding: 0 7px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 600;
    line-height: 1;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    box-sizing: border-box;
    transition: all 0.2s ease;
  }

  .tag-badge {
    background-color: #1a2333;
    color: #94a3b8;
    border: 1px solid rgba(255, 255, 255, 0.08);
    font-family: var(--font-mono, monospace);
  }

  .status-tag {
    background-color: rgba(244, 63, 94, 0.1);
    color: #fb7185;
    border: 1px solid rgba(244, 63, 94, 0.22);
  }

  .status-tag.testing {
    background-color: rgba(99, 102, 241, 0.1);
    color: #a5b4fc;
    border-color: rgba(99, 102, 241, 0.25);
  }

  .status-tag.verified {
    background-color: rgba(34, 197, 94, 0.1);
    color: #4ade80;
    border-color: rgba(34, 197, 94, 0.22);
  }

  .status-tag.error {
    background-color: rgba(239, 68, 68, 0.1);
    color: #f87171;
    border-color: rgba(239, 68, 68, 0.25);
  }

  .tag-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background-color: #f43f5e;
  }

  .tag-dot.testing {
    background-color: #818cf8;
    animation: pulse 1s infinite;
  }

  .tag-dot.verified {
    background-color: #22c55e;
  }

  .tag-dot.error {
    background-color: #f87171;
  }

  /* IP Display Row */
  .ip-value-row {
    background-color: #0b111a;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .ip-value-group {
    display: flex;
    align-items: center;
    gap: 9px;
    min-width: 0;
  }

  .globe-icon {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .ip-code {
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: #f8fafc;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: color 0.2s ease, letter-spacing 0.2s ease;
  }

  .ip-code.is-raw {
    color: #38bdf8;
    letter-spacing: 0.02em;
  }

  .ip-actions-group {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }

  .action-icon-btn {
    background: transparent;
    border: none;
    color: #94a3b8;
    padding: 4px;
    border-radius: 4px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s ease, background-color 0.15s ease;
  }

  .action-icon-btn:hover {
    color: #f8fafc;
    background-color: rgba(255, 255, 255, 0.08);
  }

  /* Bottom Hint */
  .hint-banner {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 11px;
    color: #94a3b8;
    line-height: 1.35;
  }

  .hint-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .hint-text.verified {
    color: #cbd5e1;
  }

  /* Dialog Footer */
  .dialog-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding-top: 4px;
  }

  .footer-left {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .footer-right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    flex-wrap: nowrap;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 7px 12px;
    font-size: 12px;
    font-weight: 500;
    border-radius: 8px;
    cursor: pointer;
    border: 1px solid transparent;
    line-height: 1;
    white-space: nowrap;
    transition: all 0.15s ease;
    box-sizing: border-box;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-cancel {
    background-color: rgba(244, 63, 94, 0.08);
    color: #fb7185;
    border-color: rgba(244, 63, 94, 0.2);
  }

  .btn-cancel:hover:not(:disabled) {
    background-color: rgba(244, 63, 94, 0.16);
    color: #fecdd3;
  }

  .btn-secondary {
    background-color: transparent;
    color: #94a3b8;
    border-color: transparent;
    padding: 7px 10px;
  }

  .btn-secondary:hover:not(:disabled) {
    color: #f1f5f9;
    background-color: rgba(255, 255, 255, 0.06);
  }

  .btn-secondary-action {
    background-color: #1a2333;
    color: #e2e8f0;
    border-color: rgba(255, 255, 255, 0.1);
  }

  .btn-secondary-action:hover:not(:disabled) {
    background-color: #243046;
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.16);
  }

  .btn-resume-cta {
    background-color: #151d2a;
    color: #64748b;
    border-color: rgba(255, 255, 255, 0.06);
    font-weight: 600;
    cursor: not-allowed;
  }

  .btn-resume-cta.verified {
    background-color: #15803d;
    color: #ffffff;
    border-color: #22c55e;
    cursor: pointer;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.25);
  }

  .btn-resume-cta.verified:hover:not(:disabled) {
    background-color: #16a34a;
    border-color: #4ade80;
  }

  .spin-icon {
    display: inline-flex;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>
