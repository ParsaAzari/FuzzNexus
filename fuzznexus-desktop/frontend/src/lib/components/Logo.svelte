<script lang="ts">
  import { themeStore } from '../stores/theme.svelte';

  interface Props {
    size?: number | string;
    class?: string;
    mode?: 'auto' | 'dark' | 'light';
  }

  let { size = 22, class: className = '', mode = 'auto' }: Props = $props();

  const numSize = $derived(typeof size === 'number' ? size : parseInt(String(size), 10) || 22);
  const isDark = $derived(mode === 'auto' ? themeStore.actualDark : mode === 'dark');
</script>

{#if isDark}
  <!-- Dark Mode: Obsidian Titanium Body with Illuminated Rim -->
  <svg
    width={numSize}
    height={numSize}
    viewBox="0 0 1024 1024"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    class="fuzznexus-logo fuzznexus-logo-dark {className}"
    aria-label="FuzzNexus Logo"
  >
    <defs>
      <radialGradient id="fn-darkProShadow" cx="50%" cy="50%" r="50%">
        <stop offset="65%" stop-color="#050507" stop-opacity="0.55" />
        <stop offset="85%" stop-color="#050507" stop-opacity="0.22" />
        <stop offset="100%" stop-color="#050507" stop-opacity="0" />
      </radialGradient>

      <linearGradient id="fn-darkProBody" x1="50%" y1="0%" x2="50%" y2="100%">
        <stop offset="0%" stop-color="#23252a" />
        <stop offset="55%" stop-color="#18191d" />
        <stop offset="100%" stop-color="#111215" />
      </linearGradient>

      <linearGradient id="fn-darkProRim" x1="50%" y1="0%" x2="50%" y2="100%">
        <stop offset="0%" stop-color="#ffffff" stop-opacity="0.32" />
        <stop offset="55%" stop-color="#ffffff" stop-opacity="0.10" />
        <stop offset="100%" stop-color="#000000" stop-opacity="0.6" />
      </linearGradient>

      <linearGradient id="fn-darkHandle" x1="15%" y1="15%" x2="88%" y2="88%">
        <stop offset="0%" stop-color="#2c3036" />
        <stop offset="45%" stop-color="#4b515a" />
        <stop offset="100%" stop-color="#7c838e" />
      </linearGradient>

      <linearGradient id="fn-darkLensBase" x1="20%" y1="10%" x2="80%" y2="90%">
        <stop offset="0%" stop-color="#8b929c" />
        <stop offset="50%" stop-color="#4f555e" />
        <stop offset="100%" stop-color="#727984" />
      </linearGradient>

      <linearGradient id="fn-darkLensWave" x1="15%" y1="85%" x2="85%" y2="20%">
        <stop offset="0%" stop-color="#1b1e22" />
        <stop offset="55%" stop-color="#262a30" />
        <stop offset="100%" stop-color="#363b43" />
      </linearGradient>

      <linearGradient id="fn-darkGlassHighlight" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#ffffff" stop-opacity="0.25" />
        <stop offset="100%" stop-color="#ffffff" stop-opacity="0" />
      </linearGradient>

      <linearGradient id="fn-darkGhostBody" x1="35%" y1="8%" x2="72%" y2="95%">
        <stop offset="0%" stop-color="#ffffff" />
        <stop offset="68%" stop-color="#f3f4f6" />
        <stop offset="100%" stop-color="#dde1e6" />
      </linearGradient>

      <linearGradient id="fn-darkGhostShadow" x1="0%" y1="35%" x2="100%" y2="55%">
        <stop offset="0%" stop-color="#9aa2ad" />
        <stop offset="60%" stop-color="#bcc2ca" />
        <stop offset="100%" stop-color="#f3f4f6" stop-opacity="0" />
      </linearGradient>

      <linearGradient id="fn-darkGhostEye" x1="50%" y1="0%" x2="50%" y2="100%">
        <stop offset="0%" stop-color="#181a1f" />
        <stop offset="100%" stop-color="#484e57" />
      </linearGradient>

      <clipPath id="fn-darkLensClip">
        <circle cx="476" cy="466" r="244" />
      </clipPath>
    </defs>

    <!-- Base Ambient Shadow -->
    <ellipse cx="512" cy="548" rx="444" ry="434" fill="url(#fn-darkProShadow)" />

    <!-- Squircle Container -->
    <path
      d="M 512 100
         C 716 100, 796 100, 860 164
         C 924 228, 924 308, 924 512
         C 924 716, 924 796, 860 860
         C 796 924, 716 924, 512 924
         C 308 924, 228 924, 164 860
         C 100 796, 100 716, 100 512
         C 100 308, 100 228, 164 164
         C 228 100, 308 100, 512 100 Z"
      fill="url(#fn-darkProBody)"
      stroke="url(#fn-darkProRim)"
      stroke-width="2.5"
    />

    <!-- Magnifier Handle -->
    <g transform="translate(618, 608) rotate(45)">
      <rect x="0" y="-44" width="234" height="88" rx="44" fill="url(#fn-darkHandle)" />
    </g>

    <!-- Magnifier Lens -->
    <g clip-path="url(#fn-darkLensClip)">
      <circle cx="476" cy="466" r="244" fill="url(#fn-darkLensBase)" />
      <path
        d="M 222 396
           C 214 550, 304 676, 434 676
           C 526 676, 580 598, 638 558
           C 686 525, 722 538, 732 562
           L 742 364
           C 678 308, 582 284, 518 299
           C 436 318, 402 436, 338 476
           C 284 510, 240 450, 222 396 Z"
        fill="url(#fn-darkLensWave)"
      />
      <path
        d="M 476 222
           A 244 244 0 0 0 256 574
           C 246 474, 296 340, 476 222 Z"
        fill="url(#fn-darkGlassHighlight)"
      />
    </g>

    <!-- Ghost Mark inside Lens -->
    <g id="fn-dark-ghost">
      <path
        d="M 490 342
           C 432 342, 406 382, 400 438
           C 394 494, 384 530, 362 552
           C 352 562, 356 579, 374 580
           C 402 581, 419 554, 436 554
           C 451 554, 451 575, 468 575
           C 486 575, 495 554, 514 554
           C 533 554, 539 573, 555 572
           C 569 571, 574 553, 573 531
           C 571 496, 569 456, 565 420
           C 560 371, 532 342, 490 342 Z"
        fill="url(#fn-darkGhostBody)"
      />
      <path
        d="M 471 343
           C 425 348, 405 386, 400 438
           C 394 494, 384 530, 362 552
           C 352 562, 356 579, 374 580
           C 398 581, 414 559, 430 555
           C 423 536, 423 496, 429 442
           C 434 393, 449 356, 471 343 Z"
        fill="url(#fn-darkGhostShadow)"
      />
      <ellipse cx="487" cy="418" rx="14" ry="21.5" fill="url(#fn-darkGhostEye)" />
      <ellipse cx="533" cy="417" rx="14" ry="21.5" fill="url(#fn-darkGhostEye)" />
    </g>
  </svg>
{:else}
  <!-- Light Mode: Silver Pearl Body with Crisp Edge and Subtle Shadow -->
  <svg
    width={numSize}
    height={numSize}
    viewBox="0 0 1024 1024"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    class="fuzznexus-logo fuzznexus-logo-light {className}"
    aria-label="FuzzNexus Logo"
  >
    <defs>
      <radialGradient id="fn-lightProShadow" cx="50%" cy="50%" r="50%">
        <stop offset="65%" stop-color="#000000" stop-opacity="0.14" />
        <stop offset="85%" stop-color="#000000" stop-opacity="0.05" />
        <stop offset="100%" stop-color="#000000" stop-opacity="0" />
      </radialGradient>

      <linearGradient id="fn-lightSquircle" x1="50%" y1="0%" x2="50%" y2="100%">
        <stop offset="0%" stop-color="#fbfcfd" />
        <stop offset="50%" stop-color="#f2f4f7" />
        <stop offset="100%" stop-color="#e4e7eb" />
      </linearGradient>

      <linearGradient id="fn-lightSquircleRim" x1="50%" y1="0%" x2="50%" y2="100%">
        <stop offset="0%" stop-color="#ffffff" stop-opacity="0.95" />
        <stop offset="50%" stop-color="#cbd2d9" stop-opacity="0.65" />
        <stop offset="100%" stop-color="#9aa2ad" stop-opacity="0.45" />
      </linearGradient>

      <linearGradient id="fn-lightHandle" x1="15%" y1="15%" x2="88%" y2="88%">
        <stop offset="0%" stop-color="#2e3339" />
        <stop offset="50%" stop-color="#4d535b" />
        <stop offset="100%" stop-color="#788089" />
      </linearGradient>

      <linearGradient id="fn-lightLensBase" x1="20%" y1="10%" x2="80%" y2="90%">
        <stop offset="0%" stop-color="#b8bec4" />
        <stop offset="50%" stop-color="#8e959d" />
        <stop offset="100%" stop-color="#a2a9b2" />
      </linearGradient>

      <linearGradient id="fn-lightDarkWave" x1="15%" y1="85%" x2="85%" y2="20%">
        <stop offset="0%" stop-color="#2b2f34" />
        <stop offset="50%" stop-color="#3e444b" />
        <stop offset="100%" stop-color="#535a63" />
      </linearGradient>

      <linearGradient id="fn-lightGlassHighlight" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="#ffffff" stop-opacity="0.35" />
        <stop offset="100%" stop-color="#ffffff" stop-opacity="0" />
      </linearGradient>

      <linearGradient id="fn-lightGhost" x1="35%" y1="8%" x2="72%" y2="95%">
        <stop offset="0%" stop-color="#ffffff" />
        <stop offset="68%" stop-color="#f5f6f8" />
        <stop offset="100%" stop-color="#e2e6eb" />
      </linearGradient>

      <linearGradient id="fn-lightGhostShadow" x1="0%" y1="35%" x2="100%" y2="55%">
        <stop offset="0%" stop-color="#a6aeb8" />
        <stop offset="60%" stop-color="#c5ccd5" />
        <stop offset="100%" stop-color="#f3f4f6" stop-opacity="0" />
      </linearGradient>

      <clipPath id="fn-lightLensClip">
        <circle cx="476" cy="466" r="244" />
      </clipPath>
    </defs>

    <!-- Base Ambient Shadow -->
    <ellipse cx="512" cy="548" rx="444" ry="434" fill="url(#fn-lightProShadow)" />

    <!-- Squircle Container with Rim -->
    <path
      d="M 512 100
         C 716 100, 796 100, 860 164
         C 924 228, 924 308, 924 512
         C 924 716, 924 796, 860 860
         C 796 924, 716 924, 512 924
         C 308 924, 228 924, 164 860
         C 100 796, 100 716, 100 512
         C 100 308, 100 228, 164 164
         C 228 100, 308 100, 512 100 Z"
      fill="url(#fn-lightSquircle)"
      stroke="url(#fn-lightSquircleRim)"
      stroke-width="2.5"
    />

    <!-- Magnifier Handle -->
    <g transform="translate(618, 608) rotate(45)">
      <rect x="0" y="-44" width="234" height="88" rx="44" fill="url(#fn-lightHandle)" />
    </g>

    <!-- Magnifier Lens -->
    <g clip-path="url(#fn-lightLensClip)">
      <circle cx="476" cy="466" r="244" fill="url(#fn-lightLensBase)" />
      <path
        d="M 222 396
           C 214 550, 304 676, 434 676
           C 526 676, 580 598, 638 558
           C 686 525, 722 538, 732 562
           L 742 364
           C 678 308, 582 284, 518 299
           C 436 318, 402 436, 338 476
           C 284 510, 240 450, 222 396 Z"
        fill="url(#fn-lightDarkWave)"
      />
      <path
        d="M 476 222
           A 244 244 0 0 0 256 574
           C 246 474, 296 340, 476 222 Z"
        fill="url(#fn-lightGlassHighlight)"
      />
    </g>

    <!-- Ghost Mark inside Lens -->
    <g id="fn-light-ghost">
      <path
        d="M 490 342
           C 432 342, 406 382, 400 438
           C 394 494, 384 530, 362 552
           C 352 562, 356 579, 374 580
           C 402 581, 419 554, 436 554
           C 451 554, 451 575, 468 575
           C 486 575, 495 554, 514 554
           C 533 554, 539 573, 555 572
           C 569 571, 574 553, 573 531
           C 571 496, 569 456, 565 420
           C 560 371, 532 342, 490 342 Z"
        fill="url(#fn-lightGhost)"
      />
      <path
        d="M 471 343
           C 425 348, 405 386, 400 438
           C 394 494, 384 530, 362 552
           C 352 562, 356 579, 374 580
           C 398 581, 414 559, 430 555
           C 423 536, 423 496, 429 442
           C 434 393, 449 356, 471 343 Z"
        fill="url(#fn-lightGhostShadow)"
      />
      <ellipse cx="487" cy="418" rx="14" ry="21.5" fill="#2e333a" />
      <ellipse cx="533" cy="417" rx="14" ry="21.5" fill="#2e333a" />
    </g>
  </svg>
{/if}

<style>
  .fuzznexus-logo {
    display: inline-flex;
    vertical-align: middle;
    flex-shrink: 0;
    filter: drop-shadow(0 2px 5px rgba(0, 0, 0, 0.22));
    transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), filter 0.2s ease;
  }

  .fuzznexus-logo:hover {
    transform: scale(1.05);
    filter: drop-shadow(0 3px 8px rgba(0, 0, 0, 0.32));
  }
</style>
