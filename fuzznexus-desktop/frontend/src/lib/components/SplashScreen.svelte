<script lang="ts">
  import { onMount } from 'svelte';

  let { onComplete }: { onComplete: () => void } = $props();

  let progress = $state(0);
  let statusText = $state('Initializing FuzzNexus...');
  let fadingOut = $state(false);

  const steps = [
    { pct: 24, text: 'Initializing FuzzNexus...', delay: 450 },
    { pct: 56, text: 'Loading fuzzing profiles...', delay: 850 },
    { pct: 86, text: 'Starting worker pool...', delay: 800 },
    { pct: 100, text: 'Ready', delay: 550 }
  ];

  onMount(() => {
    // Ensure launcher window controls are hidden and window is centered
    try {
      const app = (window as any).go?.main?.App;
      if (app && typeof app.HideLauncherControls === 'function') {
        app.HideLauncherControls();
      }
    } catch (_) {}

    const timers: ReturnType<typeof setTimeout>[] = [];
    let elapsed = 150;

    for (const step of steps) {
      timers.push(
        setTimeout(() => {
          progress = step.pct;
          statusText = step.text;
        }, elapsed)
      );
      elapsed += step.delay;
    }

    timers.push(
      setTimeout(() => {
        fadingOut = true;
      }, elapsed + 300)
    );

    timers.push(
      setTimeout(async () => {
        try {
          const app = (window as any).go?.main?.App;
          if (app && typeof app.FinishSplash === 'function') {
            await app.FinishSplash();
          }
        } catch (e) {
          console.warn('FinishSplash error:', e);
        }
        onComplete();
      }, elapsed + 750)
    );

    return () => {
      timers.forEach(clearTimeout);
    };
  });
</script>

<div class="splash-overlay" class:fade-out={fadingOut}>
  <div class="splash-card">
    <div class="grid-overlay"></div>

    <div class="brand-col">
      <div class="brand-header">
        <div class="product-title">FuzzNexus</div>
        <span class="version-pill">v1.0.1</span>
        <p class="tagline">Next-gen web fuzzing &amp; security reconnaissance.</p>
      </div>

      <div class="loader-col">
        <div class="status-row">
          <span class="status-label">{statusText}</span>
          <span>{progress}%</span>
        </div>
        <div class="progress-track">
          <div class="progress-fill" style="width: {progress}%"></div>
        </div>
        <div class="copyright-text">
          Copyright &copy; 2026 Parsa Azari. Licensed under the MIT License.
        </div>
      </div>
    </div>

    <div class="logo-stage">
      <div class="blue-aura"></div>
      <svg class="fuzznexus-svg" xmlns="http://www.w3.org/2000/svg" viewBox="140 140 740 740">
        <defs>
          <linearGradient id="splashHandle" x1="15%" y1="15%" x2="88%" y2="88%">
            <stop offset="0%" stop-color="#1e3a8a" />
            <stop offset="45%" stop-color="#2563eb" />
            <stop offset="100%" stop-color="#38bdf8" />
          </linearGradient>

          <linearGradient id="splashLensBase" x1="20%" y1="10%" x2="80%" y2="90%">
            <stop offset="0%" stop-color="#60a5fa" />
            <stop offset="50%" stop-color="#2563eb" />
            <stop offset="100%" stop-color="#38bdf8" />
          </linearGradient>

          <linearGradient id="splashLensWave" x1="15%" y1="85%" x2="85%" y2="20%">
            <stop offset="0%" stop-color="#0f172a" />
            <stop offset="55%" stop-color="#1e293b" />
            <stop offset="100%" stop-color="#1d4ed8" />
          </linearGradient>

          <linearGradient id="splashGlassHighlight" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#ffffff" stop-opacity="0.28" />
            <stop offset="100%" stop-color="#ffffff" stop-opacity="0" />
          </linearGradient>

          <linearGradient id="splashGhostBody" x1="35%" y1="8%" x2="72%" y2="95%">
            <stop offset="0%" stop-color="#ffffff" />
            <stop offset="68%" stop-color="#e0f2fe" />
            <stop offset="100%" stop-color="#bae6fd" />
          </linearGradient>

          <linearGradient id="splashGhostShadow" x1="0%" y1="35%" x2="100%" y2="55%">
            <stop offset="0%" stop-color="#7dd3fc" />
            <stop offset="60%" stop-color="#bae6fd" />
            <stop offset="100%" stop-color="#f3f4f6" stop-opacity="0" />
          </linearGradient>

          <linearGradient id="splashGhostEye" x1="50%" y1="0%" x2="50%" y2="100%">
            <stop offset="0%" stop-color="#0f172a" />
            <stop offset="100%" stop-color="#1e3a8a" />
          </linearGradient>

          <clipPath id="splashLensClip">
            <circle cx="476" cy="466" r="244" />
          </clipPath>
        </defs>

        <g class="magnifier-group">
          <g transform="translate(618, 608) rotate(45)">
            <rect x="0" y="-44" width="234" height="88" rx="44" fill="url(#splashHandle)" />
          </g>

          <g clip-path="url(#splashLensClip)">
            <circle cx="476" cy="466" r="244" fill="url(#splashLensBase)" />
            <path d="M 222 396 C 214 550, 304 676, 434 676 C 526 676, 580 598, 638 558 C 686 525, 722 538, 732 562 L 742 364 C 678 308, 582 284, 518 299 C 436 318, 402 436, 338 476 C 284 510, 240 450, 222 396 Z" fill="url(#splashLensWave)" />
            <path d="M 476 222 A 244 244 0 0 0 256 574 C 246 474, 296 340, 476 222 Z" fill="url(#splashGlassHighlight)" />
          </g>

          <g class="ghost-mark">
            <path d="M 490 342 C 432 342, 406 382, 400 438 C 394 494, 384 530, 362 552 C 352 562, 356 579, 374 580 C 402 581, 419 554, 436 554 C 451 554, 451 575, 468 575 C 486 575, 495 554, 514 554 C 533 554, 539 573, 555 572 C 569 571, 574 553, 573 531 C 571 496, 569 456, 565 420 C 560 371, 532 342, 490 342 Z" fill="url(#splashGhostBody)" />
            <path d="M 471 343 C 425 348, 405 386, 400 438 C 394 494, 384 530, 362 552 C 352 562, 356 579, 374 580 C 398 581, 414 559, 430 555 C 423 536, 423 496, 429 442 C 434 393, 449 356, 471 343 Z" fill="url(#splashGhostShadow)" />
            <ellipse class="ghost-eye" cx="487" cy="418" rx="14" ry="21.5" fill="url(#splashGhostEye)" />
            <ellipse class="ghost-eye" cx="533" cy="417" rx="14" ry="21.5" fill="url(#splashGhostEye)" />
          </g>
        </g>
      </svg>
    </div>
  </div>
</div>

<style>
  .splash-overlay {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: radial-gradient(circle at 50% 45%, #0f172a 0%, #020617 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Inter, sans-serif;
    color: #f8fafc;
    transition: opacity 0.45s cubic-bezier(0.16, 1, 0.3, 1), transform 0.45s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .splash-overlay.fade-out {
    opacity: 0;
    transform: scale(0.98);
    pointer-events: none;
  }

  .splash-card {
    position: relative;
    width: 600px;
    height: 315px;
    border-radius: 20px;
    background: linear-gradient(135deg, #070b14 0%, #0c1426 100%);
    border: 1px solid rgba(56, 189, 248, 0.18);
    box-shadow:
      0 28px 65px -12px rgba(0, 0, 0, 0.8),
      0 0 45px -12px rgba(59, 130, 246, 0.42);
    overflow: hidden;
    display: flex;
    align-items: center;
    padding: 34px 42px;
    animation: cardIn 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  @keyframes cardIn {
    from { opacity: 0; transform: translateY(12px) scale(0.97); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .grid-overlay {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(59, 130, 246, 0.06) 1px, transparent 1px),
      linear-gradient(90deg, rgba(59, 130, 246, 0.06) 1px, transparent 1px);
    background-size: 26px 26px;
    mask-image: radial-gradient(circle at 75% 50%, black 15%, transparent 72%);
    -webkit-mask-image: radial-gradient(circle at 75% 50%, black 15%, transparent 72%);
  }

  .brand-col {
    position: relative;
    z-index: 2;
    width: 56%;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }

  .brand-header {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .product-title {
    font-size: 32px;
    font-weight: 800;
    letter-spacing: -0.6px;
    background: linear-gradient(90deg, #ffffff 0%, #bae6fd 100%);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .version-pill {
    align-self: flex-start;
    font-size: 11px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    color: #38bdf8;
    background: rgba(56, 189, 248, 0.1);
    border: 1px solid rgba(56, 189, 248, 0.25);
    padding: 3px 10px;
    border-radius: 999px;
    letter-spacing: 0.4px;
  }

  .tagline {
    font-size: 13px;
    color: #94a3b8;
    line-height: 1.5;
    margin-top: 2px;
  }

  .loader-col { width: 100%; }

  .status-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    color: #94a3b8;
    margin-bottom: 8px;
  }

  .status-label { color: #e2e8f0; }

  .progress-track {
    width: 100%;
    height: 4px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 999px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, #2563eb, #38bdf8);
    box-shadow: 0 0 12px rgba(59, 130, 246, 0.42);
    transition: width 0.28s ease-out;
  }

  .copyright-text {
    margin-top: 10px;
    font-size: 10.5px;
    color: #64748b;
    letter-spacing: 0.2px;
  }

  .logo-stage {
    position: absolute;
    right: 14px;
    top: 50%;
    transform: translateY(-50%);
    width: 246px;
    height: 246px;
    z-index: 1;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .blue-aura {
    position: absolute;
    width: 186px;
    height: 186px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(56, 189, 248, 0.28) 0%, rgba(37, 99, 235, 0.08) 55%, transparent 75%);
    animation: auraPulse 2.8s ease-in-out infinite;
  }

  @keyframes auraPulse {
    0%, 100% { transform: scale(0.92); opacity: 0.7; }
    50% { transform: scale(1.12); opacity: 1; }
  }

  .fuzznexus-svg {
    width: 226px;
    height: 226px;
    overflow: visible;
    filter: drop-shadow(0 12px 24px rgba(0, 0, 0, 0.55));
  }

  .magnifier-group {
    transform-origin: 476px 466px;
    animation: scanMotion 3.6s ease-in-out infinite;
  }

  .ghost-mark {
    transform-origin: 476px 466px;
    animation: ghostFloat 2.6s ease-in-out infinite;
  }

  .ghost-eye {
    transform-origin: center;
    transform-box: fill-box;
    animation: eyeBlink 3.4s ease-in-out infinite;
  }

  @keyframes scanMotion {
    0%, 100% { transform: rotate(-4deg) translate(0px, 0px); }
    50% { transform: rotate(5deg) translate(6px, -6px); }
  }

  @keyframes ghostFloat {
    0%, 100% { transform: translateY(0px); }
    50% { transform: translateY(-14px); }
  }

  @keyframes eyeBlink {
    0%, 44%, 52%, 100% { transform: scaleY(1); }
    48% { transform: scaleY(0.12); }
  }
</style>
