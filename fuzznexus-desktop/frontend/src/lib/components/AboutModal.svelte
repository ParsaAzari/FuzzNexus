<script lang="ts">
  import Icon from './Icon.svelte';
  import Logo from './Logo.svelte';

  interface Props {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: Props = $props();

  function openLink(url: string) {
    if (typeof window !== 'undefined') {
      const goApp = (window as unknown as { go?: { main?: { App?: { OpenURL?: (url: string) => Promise<void> } } } })?.go?.main?.App;
      if (goApp?.OpenURL) {
        goApp.OpenURL(url);
        return;
      }
      const runtime = (window as unknown as { runtime?: { BrowserOpenURL?: (url: string) => void } })?.runtime;
      if (runtime?.BrowserOpenURL) {
        runtime.BrowserOpenURL(url);
        return;
      }
      window.open(url, '_blank');
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen) {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div
    class="about-backdrop"
    onclick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}
    role="presentation"
  >
    <div
      class="about-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="about-title"
    >
      <!-- Close Button -->
      <button
        type="button"
        class="about-close-btn"
        onclick={onClose}
        aria-label="Close About Dialog"
        style="--wails-draggable: no-drag;"
      >
        <Icon name="x" size={16} />
      </button>

      <!-- Hero Header with Native Window Dragging -->
      <div class="about-hero" style="--wails-draggable: drag;">
        <div class="logo-wrapper" style="--wails-draggable: drag;">
          <Logo size={72} />
        </div>
        <div class="about-titles" style="--wails-draggable: drag;">
          <div class="title-row" style="--wails-draggable: drag;">
            <h2 id="about-title" class="app-brand-name">FuzzNexus</h2>
            <span class="version-badge font-mono">v1.0.2</span>
          </div>
          <p class="app-tagline">
            High-Performance Autonomous Web Fuzzer & Security Engine
          </p>
          <div class="developer-tag">
            <span>Developed by</span>
            <strong class="developer-name">Parsa Azari</strong>
          </div>
        </div>
      </div>

      <!-- Divider -->
      <div class="about-divider"></div>

      <!-- Links Grid -->
      <div class="about-links-section" style="--wails-draggable: no-drag;">
        <span class="section-heading">Official Channels & Links</span>

        <div class="links-grid">
          <!-- Website Card -->
          <button
            type="button"
            class="link-card"
            onclick={() => openLink('https://parsaazari.com/')}
            title="Open parsaazari.com in browser"
          >
            <div class="link-icon-box">
              <Icon name="globe" size={20} color="var(--accent)" />
            </div>
            <div class="link-info">
              <div class="link-title-wrap">
                <span class="link-title font-mono">parsaazari.com</span>
                <Icon name="external-link" size={12} class="link-arrow" />
              </div>
              <span class="link-desc">Official Website, Portfolio & Research</span>
            </div>
          </button>

          <!-- GitHub Card -->
          <button
            type="button"
            class="link-card"
            onclick={() => openLink('https://github.com/ParsaAzari/FuzzNexus')}
            title="Open github.com/ParsaAzari/FuzzNexus in browser"
          >
            <div class="link-icon-box github-icon-box">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
              </svg>
            </div>
            <div class="link-info">
              <div class="link-title-wrap">
                <span class="link-title font-mono">github.com/ParsaAzari/FuzzNexus</span>
                <Icon name="external-link" size={12} class="link-arrow" />
              </div>
              <span class="link-desc">Source Code, Releases &amp; Documentation</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Specs & System Badges -->
      <div class="about-specs-wrap">
        <div class="specs-grid">
          <div class="spec-pill">
            <span class="spec-key">Core</span>
            <span class="spec-val">Go 1.26 Engine</span>
          </div>
          <div class="spec-pill">
            <span class="spec-key">Framework</span>
            <span class="spec-val">Wails v2 Desktop</span>
          </div>
          <div class="spec-pill">
            <span class="spec-key">Frontend</span>
            <span class="spec-val">Svelte 5</span>
          </div>
          <div class="spec-pill">
            <span class="spec-key">License</span>
            <span class="spec-val">MIT License</span>
          </div>
        </div>
      </div>

      <!-- Footer Copyright -->
      <div class="about-footer">
        <p class="copyright-text">
          Copyright 2026 Parsa Azari. Licensed under the MIT License.
        </p>
      </div>
    </div>
  </div>
{/if}

<style>
  .about-backdrop {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    transform: translateZ(0);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    animation: fadeIn 0.18s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .about-dialog {
    position: relative;
    width: 100%;
    max-width: 544px;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 14px);
    box-shadow: 0 24px 48px -12px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 255, 255, 0.08);
    padding: 28px;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 20px;
    animation: scaleIn 0.22s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes scaleIn {
    from {
      opacity: 0;
      transform: scale(0.96) translateY(8px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .about-close-btn {
    position: absolute;
    top: 16px;
    right: 16px;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 1px solid var(--border);
    background: var(--bg-subtle);
    color: var(--text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .about-close-btn:hover {
    background: var(--border);
    color: var(--text);
    transform: scale(1.05);
  }

  .about-hero {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .logo-wrapper {
    flex-shrink: 0;
    filter: drop-shadow(0 6px 16px rgba(0, 0, 0, 0.35));
  }

  .about-titles {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .app-brand-name {
    margin: 0;
    font-size: 24px;
    font-weight: 800;
    letter-spacing: -0.03em;
    color: var(--text);
    line-height: 1.1;
  }

  .version-badge {
    font-size: 11px;
    font-weight: 700;
    color: var(--accent);
    background: rgba(0, 122, 255, 0.12);
    border: 1px solid rgba(0, 122, 255, 0.25);
    padding: 2px 7px;
    border-radius: 9999px;
  }

  .app-tagline {
    margin: 0;
    font-size: 13px;
    line-height: 1.4;
    color: var(--text-muted);
  }

  .developer-tag {
    display: flex;
    align-items: baseline;
    gap: 5px;
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 2px;
  }

  .developer-name {
    color: var(--text);
    font-weight: 600;
  }

  .about-divider {
    height: 1px;
    background: var(--border);
    width: 100%;
  }

  .about-links-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .section-heading {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }

  .links-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .link-card {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 14px;
    background: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-md, 8px);
    text-align: left;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
    color: inherit;
    width: 100%;
    box-sizing: border-box;
  }

  .link-card:hover {
    background: var(--bg);
    border-color: var(--accent);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }

  .link-card:active {
    transform: translateY(0);
  }

  .link-icon-box {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    background: rgba(0, 122, 255, 0.08);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s ease;
  }

  .github-icon-box {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text);
  }

  .link-card:hover .link-icon-box {
    background: rgba(0, 122, 255, 0.16);
  }

  .link-card:hover .github-icon-box {
    background: rgba(255, 255, 255, 0.12);
  }

  .link-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    flex: 1;
  }

  .link-title-wrap {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .link-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }

  :global(.link-arrow) {
    color: var(--text-muted);
    transition: transform 0.15s ease, color 0.15s ease;
  }

  .link-card:hover :global(.link-arrow) {
    color: var(--accent);
    transform: translate(2px, -2px);
  }

  .link-desc {
    font-size: 12px;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .about-specs-wrap {
    margin-top: -4px;
  }

  .specs-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .spec-pill {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
    min-width: 0;
    padding: 7px 12px;
    background: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 11px;
  }

  .spec-key {
    color: var(--text-muted);
    flex-shrink: 0;
  }

  .spec-val {
    font-weight: 600;
    color: var(--text);
    font-family: var(--font-mono, monospace);
    white-space: nowrap;
    text-align: right;
  }

  .about-footer {
    display: flex;
    align-items: center;
    justify-content: center;
    padding-top: 4px;
  }

  .copyright-text {
    margin: 0;
    font-size: 11px;
    color: var(--text-muted);
    text-align: center;
  }
</style>
