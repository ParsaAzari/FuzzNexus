<script lang="ts">
  import { resultsStore } from '../stores/results.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { GetResultPayload } from '../../../wailsjs/go/main/App';
  import type { PayloadDTO } from '../types';
  import Icon from './Icon.svelte';

  type TabType = 'response' | 'request' | 'metadata';
  type ViewMode = 'raw' | 'preview';

  let activeTab = $state<TabType>('response');
  let viewMode = $state<ViewMode>('raw');
  let payload = $state<PayloadDTO | null>(null);
  let isLoading = $state<boolean>(false);
  let loadError = $state<string>('');
  let copied = $state<boolean>(false);
  let copiedField = $state<string>('');

  // Beautifier & Formatting state (Defaults to true for instant IDE-style readability)
  let isPretty = $state<boolean>(true);

  // Live in-response search state
  let isSearchOpen = $state<boolean>(false);
  let searchQuery = $state<string>('');
  let activeMatchIndex = $state<number>(0);
  let searchInputEl = $state<HTMLInputElement | null>(null);
  let codeScrollContainer = $state<HTMLDivElement | null>(null);

  let selectedItem = $derived(resultsStore.selectedItem);

  $effect(() => {
    const item = selectedItem;
    const sid = scanStore.sessionID || resultsStore.sessionID;
    if (item && sid > 0) {
      loadPayload(sid, item.id);
    } else {
      payload = null;
      loadError = '';
    }
  });

  async function loadPayload(sessionID: number, resultID: number) {
    isLoading = true;
    loadError = '';
    try {
      const data = await GetResultPayload(sessionID, resultID, 500 * 1024);
      payload = data;
    } catch (err: any) {
      loadError = err?.message || String(err);
      payload = null;
    } finally {
      isLoading = false;
    }
  }

  function copyToClipboard(text: string, fieldName = '') {
    if (!text) return;
    navigator.clipboard.writeText(text).then(() => {
      if (fieldName) {
        copiedField = fieldName;
        setTimeout(() => (copiedField = ''), 2000);
      } else {
        copied = true;
        setTimeout(() => (copied = false), 2000);
      }
    });
  }

  function getStatusCategory(status: number): '2xx' | '3xx' | '4xx' | '5xx' | 'err' {
    if (status >= 200 && status < 300) return '2xx';
    if (status >= 300 && status < 400) return '3xx';
    if (status >= 400 && status < 500) return '4xx';
    if (status >= 500 && status < 600) return '5xx';
    return 'err';
  }

  function getStatusBadgeClass(status: number): string {
    return `badge-${getStatusCategory(status)}`;
  }

  function getStatusReason(status: number): string {
    const reasons: Record<number, string> = {
      200: 'OK',
      201: 'Created',
      204: 'No Content',
      301: 'Moved Permanently',
      302: 'Found',
      304: 'Not Modified',
      307: 'Temporary Redirect',
      308: 'Permanent Redirect',
      400: 'Bad Request',
      401: 'Unauthorized',
      403: 'Forbidden',
      404: 'Not Found',
      405: 'Method Not Allowed',
      429: 'Too Many Requests',
      500: 'Internal Server Error',
      502: 'Bad Gateway',
      503: 'Service Unavailable',
      504: 'Gateway Timeout'
    };
    return reasons[status] || '';
  }

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  }

  // --- String, Formatting & Syntax Utilities ---

  function escapeHTML(str: string): string {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function escapeRegex(str: string): string {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  function detectLanguage(contentType: string, text: string): 'json' | 'html' | 'xml' | 'http' | 'text' {
    const ct = (contentType || '').toLowerCase();
    if (ct.includes('application/json') || ct.includes('+json') || ct.includes('/json')) return 'json';
    if (ct.includes('text/html') || ct.includes('application/xhtml')) return 'html';
    if (ct.includes('xml')) return 'xml';

    const trimmed = text.trim();
    if (!trimmed) return 'text';

    if (/^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|CONNECT|TRACE)\s+\S+\s+HTTP\//i.test(trimmed)) {
      return 'http';
    }

    if ((trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
      try {
        JSON.parse(trimmed);
        return 'json';
      } catch {}
    }
    if (/^[\{\[]\s*(".*?"|\d+|true|false|null)/s.test(trimmed)) {
      return 'json';
    }

    const lower = trimmed.toLowerCase();
    if (
      lower.startsWith('<!doctype html') ||
      lower.startsWith('<html') ||
      lower.startsWith('<head') ||
      lower.startsWith('<body') ||
      /<\/?(html|head|body|div|p|span|table|form|input|button|script|style|title|meta|link|a)\b/i.test(trimmed)
    ) {
      return 'html';
    }
    if (lower.startsWith('<?xml') || /<[a-zA-Z0-9_-]+(\s+[^>]*)?>[\s\S]*<\/[a-zA-Z0-9_-]+>/s.test(trimmed)) {
      return 'xml';
    }
    return 'text';
  }

  function formatCSS(css: string, baseIndent = '  '): string {
    let clean = css.replace(/\s+/g, ' ').trim();
    clean = clean.replace(/([^;{}]+)\s*\}/g, '$1;\n}');
    clean = clean
      .replace(/\s*\{\s*/g, ' {\n')
      .replace(/\s*;\s*/g, ';\n')
      .replace(/\s*\}\s*/g, '\n}\n');

    const rawLines = clean.split('\n');
    let result: string[] = [];
    let indentLevel = 0;
    const pad = '  ';

    for (let line of rawLines) {
      line = line.trim();
      if (!line) continue;
      if (line === '}') {
        indentLevel = Math.max(0, indentLevel - 1);
        result.push(baseIndent + pad.repeat(indentLevel) + line);
      } else {
        result.push(baseIndent + pad.repeat(indentLevel) + line);
        if (line.endsWith('{')) {
          indentLevel++;
        }
      }
    }
    return result.join('\n');
  }

  function formatHTML(html: string): string {
    let output = '';
    let indent = 0;
    const pad = '  ';
    const voidTags = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'param', 'source', 'track', 'wbr']);

    // Extract and format style blocks separately so nested CSS rules are properly indented
    const styleBlocks: { attrs: string; content: string }[] = [];
    const styleBlockRe = new RegExp('<style([^>]*)>([\\s\\S]*?)<\\/' + 'style>', 'gi');

    const preprocessed = html.replace(styleBlockRe, (_, attrs, content) => {
      const idx = styleBlocks.length;
      styleBlocks.push({ attrs, content });
      return '<style-placeholder-' + idx + '>';
    });

    const tokens = preprocessed.replace(/>\s*</g, '><').match(/<[^>]+>|[^<]+/g) || [preprocessed];

    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i].trim();
      if (!token) continue;

      const styleMatch = token.match(/^<style-placeholder-(\d+)>$/);
      if (styleMatch) {
        const { attrs, content } = styleBlocks[parseInt(styleMatch[1], 10)];
        const styleIndent = pad.repeat(indent);
        const innerIndent = pad.repeat(indent + 1);
        const formattedCSS = formatCSS(content, innerIndent);
        output += styleIndent + '<style' + attrs + '>\n';
        if (formattedCSS.trim()) {
          output += formattedCSS + '\n';
        }
        output += styleIndent + '</' + 'style>\n';
        continue;
      }

      if (token.startsWith('</')) {
        indent = Math.max(0, indent - 1);
        output += pad.repeat(indent) + token + '\n';
      } else if (token.endsWith('/>') || token.startsWith('<!--') || token.startsWith('<!') || token.startsWith('<?')) {
        output += pad.repeat(indent) + token + '\n';
      } else if (token.startsWith('<')) {
        output += pad.repeat(indent) + token + '\n';
        const m = token.match(/<([a-zA-Z0-9_-]+)/);
        const tagName = m ? m[1].toLowerCase() : '';
        if (!voidTags.has(tagName)) {
          indent++;
        }
      } else {
        output += pad.repeat(indent) + token + '\n';
      }
    }
    return output.trimEnd();
  }

  function beautifyContent(text: string, lang: 'json' | 'html' | 'xml' | 'http' | 'text'): string {
    if (!text) return '';
    if (lang === 'json') {
      try {
        const parsed = JSON.parse(text);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return text;
      }
    }
    if (lang === 'html' || lang === 'xml') {
      return formatHTML(text);
    }
    return text;
  }

  function highlightJSON(raw: string): string {
    const jsonTokenRegex = /("(?:\\.|[^"\\])*")(\s*:)?|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b|\b(true|false|null)\b/g;
    let lastIndex = 0;
    let result = '';
    let match: RegExpExecArray | null;

    while ((match = jsonTokenRegex.exec(raw)) !== null) {
      result += escapeHTML(raw.slice(lastIndex, match.index));
      lastIndex = jsonTokenRegex.lastIndex;

      const [_, str, colon, num, boolVal] = match;
      if (str) {
        if (colon) {
          result += `<span class="tok-key">${escapeHTML(str)}</span>${colon}`;
        } else {
          result += `<span class="tok-str">${escapeHTML(str)}</span>`;
        }
      } else if (num !== undefined) {
        result += `<span class="tok-num">${escapeHTML(num)}</span>`;
      } else if (boolVal !== undefined) {
        result += `<span class="tok-bool">${escapeHTML(boolVal)}</span>`;
      }
    }
    result += escapeHTML(raw.slice(lastIndex));
    return result;
  }

  function highlightHTMLAttrs(rawAttrs: string): string {
    const attrRegex = /([a-zA-Z0-9_:-]+)(?:(\s*=\s*)(".*?"|'.*?'|[^\s"'=<>`]+))?/g;
    let lastIndex = 0;
    let result = '';
    let match: RegExpExecArray | null;

    while ((match = attrRegex.exec(rawAttrs)) !== null) {
      result += escapeHTML(rawAttrs.slice(lastIndex, match.index));
      lastIndex = attrRegex.lastIndex;

      const [_, name, eq, val] = match;
      result += `<span class="tok-attr">${escapeHTML(name)}</span>`;
      if (eq) {
        result += eq;
        if (val) {
          result += `<span class="tok-str">${escapeHTML(val)}</span>`;
        }
      }
    }
    result += escapeHTML(rawAttrs.slice(lastIndex));
    return result;
  }

  function highlightHTML(raw: string): string {
    const htmlTagRegex = /(<!--[\s\S]*?-->)|(<!DOCTYPE[\s\S]*?>)|(<\/?)([a-zA-Z0-9:-]+)((?:[^>"']|"[^"]*"|'[^']*')*?)(\/?>)/gi;
    let lastIndex = 0;
    let result = '';
    let match: RegExpExecArray | null;

    while ((match = htmlTagRegex.exec(raw)) !== null) {
      const textBetween = raw.slice(lastIndex, match.index);
      if (textBetween) {
        let escText = escapeHTML(textBetween)
          .replace(/(&amp;[a-zA-Z0-9#]+;)/g, '<span class="tok-ent">$1</span>')
          .replace(/^(\s*)([a-zA-Z-]+)(\s*:\s*)([^;<>{}\n]+)(;?)/gm, (_, sp, p, c, v, s) => {
            return `${sp}<span class="tok-prop">${p}</span>${c}<span class="tok-val">${v}</span>${s}`;
          })
          .replace(/^(\s*)([.#a-zA-Z0-9_:\s,-]+?)(\s*\{)$/gm, (m, sp, sel, br) => {
            if (sel.includes('&lt;') || sel.includes('&gt;')) return m;
            return `${sp}<span class="tok-sel">${sel}</span>${br}`;
          });
        result += escText;
      }
      lastIndex = htmlTagRegex.lastIndex;

      const [_, comment, doctype, tagPrefix, tagName, rawAttrs, tagSuffix] = match;
      if (comment) {
        result += `<span class="tok-comm">${escapeHTML(comment)}</span>`;
      } else if (doctype) {
        result += `<span class="tok-doctype">${escapeHTML(doctype)}</span>`;
      } else if (tagName) {
        const escPrefix = escapeHTML(tagPrefix);
        const escSuffix = escapeHTML(tagSuffix);
        result += `${escPrefix}<span class="tok-tag">${escapeHTML(tagName)}</span>`;
        if (rawAttrs && rawAttrs.trim()) {
          result += highlightHTMLAttrs(rawAttrs);
        } else if (rawAttrs) {
          result += rawAttrs;
        }
        result += escSuffix;
      }
    }

    const trailing = raw.slice(lastIndex);
    if (trailing) {
      result += escapeHTML(trailing)
        .replace(/(&amp;[a-zA-Z0-9#]+;)/g, '<span class="tok-ent">$1</span>')
        .replace(/^(\s*)([a-zA-Z-]+)(\s*:\s*)([^;<>{}\n]+)(;?)/gm, (_, sp, p, c, v, s) => {
          return `${sp}<span class="tok-prop">${p}</span>${c}<span class="tok-val">${v}</span>${s}`;
        })
        .replace(/^(\s*)([.#a-zA-Z0-9_:\s,-]+?)(\s*\{)$/gm, (m, sp, sel, br) => {
          if (sel.includes('&lt;') || sel.includes('&gt;')) return m;
          return `${sp}<span class="tok-sel">${sel}</span>${br}`;
        });
    }

    return result;
  }

  function highlightHTTP(raw: string): string {
    if (!raw) return '';

    const splitIdx = raw.indexOf('\r\n\r\n') !== -1 ? raw.indexOf('\r\n\r\n') : raw.indexOf('\n\n');
    const hasBody = splitIdx !== -1;
    const headersPart = hasBody ? raw.slice(0, splitIdx) : raw;
    const bodyPart = hasBody ? raw.slice(splitIdx + (raw.indexOf('\r\n\r\n') !== -1 ? 4 : 2)) : '';
    const sep = hasBody ? (raw.indexOf('\r\n\r\n') !== -1 ? '\r\n\r\n' : '\n\n') : '';

    const lines = headersPart.split(/\r?\n/);
    const formattedHeaders: string[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (i === 0) {
        // HTTP Request Line: METHOD PATH HTTP/VERSION
        const reqLineMatch = line.match(/^([A-Z]+)\s+(\S+)(?:\s+(HTTP\/[0-9.]+))?$/);
        if (reqLineMatch) {
          const [_, method, path, proto] = reqLineMatch;
          let l = `<span class="tok-tag">${escapeHTML(method)}</span> <span class="tok-attr">${escapeHTML(path)}</span>`;
          if (proto) {
            l += ` <span class="tok-doctype">${escapeHTML(proto)}</span>`;
          }
          formattedHeaders.push(l);
          continue;
        }
      }

      // HTTP Header Line: Key: Value
      const headerMatch = line.match(/^([a-zA-Z0-9_-]+)(\s*:\s*)(.*)$/);
      if (headerMatch) {
        const [_, name, colon, val] = headerMatch;
        formattedHeaders.push(`<span class="tok-prop">${escapeHTML(name)}</span>${colon}<span class="tok-val">${escapeHTML(val)}</span>`);
      } else {
        formattedHeaders.push(escapeHTML(line));
      }
    }

    let result = formattedHeaders.join('\n');
    if (sep) {
      result += '\n\n';
      if (bodyPart.trim()) {
        const trimmedBody = bodyPart.trim();
        if ((trimmedBody.startsWith('{') && trimmedBody.endsWith('}')) || (trimmedBody.startsWith('[') && trimmedBody.endsWith(']'))) {
          result += highlightJSON(bodyPart);
        } else if (trimmedBody.startsWith('<')) {
          result += highlightHTML(bodyPart);
        } else {
          result += escapeHTML(bodyPart);
        }
      }
    }

    return result;
  }

  function highlightTokens(raw: string, lang: 'json' | 'html' | 'xml' | 'http' | 'text'): string {
    if (!raw) return '';
    if (lang === 'json') {
      return highlightJSON(raw);
    }
    if (lang === 'html' || lang === 'xml') {
      return highlightHTML(raw);
    }
    if (lang === 'http') {
      return highlightHTTP(raw);
    }
    return escapeHTML(raw);
  }

  // Active raw text for the current tab
  let currentRawText = $derived.by(() => {
    if (activeTab === 'response') return payload?.response_text || '';
    if (activeTab === 'request') return payload?.request_text || '';
    return '';
  });

  let detectedLang = $derived.by(() => {
    if (activeTab === 'request') {
      return 'http';
    }
    const serverHeader = selectedItem?.content_type || '';
    const sniffedType = payload?.content_type || '';
    const ct = activeTab === 'response'
      ? (serverHeader && !serverHeader.includes('text/plain') ? serverHeader : (sniffedType || serverHeader || 'text/plain'))
      : 'text/plain';
    return detectLanguage(ct, currentRawText);
  });

  // Effective text to display (raw vs beautified)
  let processedText = $derived.by(() => {
    if (!currentRawText) return '';
    if (activeTab === 'response' && isPretty) {
      return beautifyContent(currentRawText, detectedLang);
    }
    return currentRawText;
  });

  // Line numbers gutter string
  let gutterString = $derived.by(() => {
    if (!processedText) return '1';
    const lines = processedText.split('\n');
    const totalLines = Math.max(1, lines.length);
    // Pre-calculate line numbers block
    return Array.from({ length: totalLines }, (_, i) => i + 1).join('\n');
  });

  // Highlighted HTML with search query injection
  let highlightedContent = $derived.by(() => {
    const raw = processedText;
    if (!raw) return { html: '', matchCount: 0 };

    // Apply syntax tokens for payloads up to 2MB (GetResultPayload caps at 500KB)
    let styledHTML = raw.length <= 2000000 ? highlightTokens(raw, detectedLang) : escapeHTML(raw);

    // Apply search highlighting if query is active
    const q = searchQuery.trim();
    if (!q) {
      return { html: styledHTML, matchCount: 0 };
    }

    const escapedQ = escapeHTML(q);
    const regexQ = escapeRegex(escapedQ);
    const re = new RegExp(`(${regexQ})(?![^<]*>)`, 'gi');

    let count = 0;
    const finalHTML = styledHTML.replace(re, (m) => {
      const idx = count++;
      const isActive = idx === activeMatchIndex;
      return `<mark id="search-match-${idx}" class="search-match ${isActive ? 'active-match' : ''}">${m}</mark>`;
    });

    return { html: finalHTML, matchCount: count };
  });

  // Live match count
  let totalMatches = $derived(highlightedContent.matchCount);

  // --- Search Navigation & Hotkeys ---

  function openSearch() {
    isSearchOpen = true;
    setTimeout(() => {
      searchInputEl?.focus();
      searchInputEl?.select();
    }, 40);
  }

  function closeSearch() {
    isSearchOpen = false;
    searchQuery = '';
    activeMatchIndex = 0;
  }

  function nextMatch() {
    if (totalMatches === 0) return;
    activeMatchIndex = (activeMatchIndex + 1) % totalMatches;
    scrollToActiveMatch();
  }

  function prevMatch() {
    if (totalMatches === 0) return;
    activeMatchIndex = (activeMatchIndex - 1 + totalMatches) % totalMatches;
    scrollToActiveMatch();
  }

  function scrollToActiveMatch() {
    setTimeout(() => {
      const el = document.getElementById(`search-match-${activeMatchIndex}`);
      if (el) {
        el.scrollIntoView({ block: 'center', behavior: 'smooth' });
      }
    }, 30);
  }

  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (e.shiftKey) prevMatch();
      else nextMatch();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      closeSearch();
    }
  }

  function handleInspectorKeydown(e: KeyboardEvent) {
    const isMeta = e.metaKey || e.ctrlKey;
    if (isMeta && e.key.toLowerCase() === 'f') {
      if (activeTab === 'response' || activeTab === 'request') {
        e.preventDefault();
        e.stopPropagation();
        openSearch();
      }
    } else if (e.key === 'Escape' && isSearchOpen) {
      e.preventDefault();
      e.stopPropagation();
      closeSearch();
    }
  }

  // Construct secure srcdoc for sandboxed iframe
  let previewSrcdoc = $derived.by(() => {
    if (!payload || !payload.response_text) return '';
    const body = payload.response_text;
    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline';">
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      font-size: 13px;
      line-height: 1.5;
      padding: 14px;
      margin: 0;
      color: #1d1d1f;
      background-color: #ffffff;
      word-break: break-word;
    }
  </style>
</head>
<body>
${body}
</body>
</html>`;
  });
</script>

<svelte:window onkeydown={handleInspectorKeydown} />

<div class="inspector-wrapper select-none">
  {#if !selectedItem}
    <div class="empty-inspector">
      <div class="empty-icon">
        <Icon name="file-text" size={36} />
      </div>
      <div class="empty-text">Select a row from the grid to inspect HTTP request & response details</div>
    </div>
  {:else}
    <!-- Inspector Header Bar (Locked 3-Column Grid) -->
    <div class="inspector-header">
      <div class="header-left">
        <span class="inspector-title">Inspector</span>
        <span class="item-badge font-mono">#{selectedItem.id}</span>
        <span class="url-badge font-mono" title={selectedItem.url}>{selectedItem.url}</span>
      </div>

      <!-- Segmented Tab Switcher (Anchored in Absolute Center) -->
      <div class="tab-group" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'response'}
          class="tab-btn"
          class:active={activeTab === 'response'}
          onclick={() => {
            activeTab = 'response';
            if (isSearchOpen) closeSearch();
          }}
        >
          Response
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'request'}
          class="tab-btn"
          class:active={activeTab === 'request'}
          onclick={() => {
            activeTab = 'request';
            if (isSearchOpen) closeSearch();
          }}
        >
          Request
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'metadata'}
          class="tab-btn"
          class:active={activeTab === 'metadata'}
          onclick={() => {
            activeTab = 'metadata';
            if (isSearchOpen) closeSearch();
          }}
        >
          Metadata
        </button>
      </div>

      <!-- Header Actions (Right Aligned) -->
      <div class="header-right">
        {#if activeTab === 'response'}
          <!-- Raw / Preview Switcher -->
          <div class="view-mode-toggle">
            <button
              type="button"
              class="toggle-btn"
              class:active={viewMode === 'raw'}
              onclick={() => (viewMode = 'raw')}
            >
              Raw
            </button>
            <button
              type="button"
              class="toggle-btn"
              class:active={viewMode === 'preview'}
              onclick={() => {
                viewMode = 'preview';
                if (isSearchOpen) closeSearch();
              }}
              title="Render HTML in sandboxed iframe"
            >
              Preview
            </button>
          </div>

          <!-- Beautify / Format Toggle (Active only in Raw mode) -->
          {#if viewMode === 'raw'}
            <button
              type="button"
              class="btn-beautify"
              class:active={isPretty}
              onclick={() => (isPretty = !isPretty)}
              title={isPretty ? "Display original raw wire payload" : "Beautify & indent response payload"}
              aria-label="Toggle beautifier"
            >
              <Icon name="code" size={12} />
              <span>Beautify</span>
            </button>
          {/if}
        {/if}

        <!-- Find in Text Button (Cmd+F / Ctrl+F) -->
        {#if viewMode === 'raw' && (activeTab === 'response' || activeTab === 'request')}
          <button
            type="button"
            class="btn-search-trigger"
            class:active={isSearchOpen}
            onclick={() => {
              if (isSearchOpen) closeSearch();
              else openSearch();
            }}
            title="Find in text (Cmd+F / Ctrl+F)"
            aria-label="Find in text"
          >
            <Icon name="search" size={13} />
          </button>
        {/if}

        <button
          type="button"
          class="btn-copy"
          onclick={() => {
            if (activeTab === 'response') copyToClipboard(processedText || payload?.response_text || '');
            else if (activeTab === 'request') copyToClipboard(payload?.request_text || '');
            else copyToClipboard(JSON.stringify(selectedItem, null, 2));
          }}
          aria-label="Copy active tab content"
        >
          {#if copied}
            <Icon name="check" size={13} />
            <span>Copied</span>
          {:else}
            <Icon name="copy" size={13} />
            <span>Copy</span>
          {/if}
        </button>

        <button
          type="button"
          class="btn-close"
          onclick={() => resultsStore.select(null)}
          title="Close Inspector"
          aria-label="Close Inspector"
        >
          <Icon name="x" size={14} />
        </button>
      </div>
    </div>

    <!-- Inspector Content Area -->
    <div class="inspector-content">
      <!-- Floating Live Search Widget -->
      {#if isSearchOpen && viewMode === 'raw' && (activeTab === 'response' || activeTab === 'request')}
        <div class="search-widget" role="search">
          <div class="search-input-wrap">
            <Icon name="search" size={12} />
            <input
              bind:this={searchInputEl}
              type="text"
              class="search-input font-mono"
              placeholder="Find in response..."
              bind:value={searchQuery}
              onkeydown={handleSearchKeydown}
              aria-label="Find in response text"
            />
          </div>

          <span class="match-badge font-mono" class:no-match={searchQuery && totalMatches === 0}>
            {#if !searchQuery}
              0 of 0
            {:else if totalMatches === 0}
              No matches
            {:else}
              {activeMatchIndex + 1} of {totalMatches}
            {/if}
          </span>

          <div class="search-nav-btns">
            <button
              type="button"
              class="search-nav-btn"
              onclick={prevMatch}
              disabled={totalMatches === 0}
              title="Previous match (Shift+Enter)"
              aria-label="Previous match"
            >
              <Icon name="arrow-up" size={12} />
            </button>
            <button
              type="button"
              class="search-nav-btn"
              onclick={nextMatch}
              disabled={totalMatches === 0}
              title="Next match (Enter)"
              aria-label="Next match"
            >
              <Icon name="arrow-down" size={12} />
            </button>
          </div>

          <button
            type="button"
            class="search-close-btn"
            onclick={closeSearch}
            title="Close search (Escape)"
            aria-label="Close search"
          >
            <Icon name="x" size={12} />
          </button>
        </div>
      {/if}

      {#if isLoading}
        <div class="loading-state">
          <div class="spinner"></div>
          <span>Loading payload...</span>
        </div>
      {:else if loadError}
        <div class="error-state">
          <Icon name="alert-triangle" size={16} />
          <span>Failed to load payload: {loadError}</span>
        </div>
      {:else if activeTab === 'response'}
        {#if payload?.truncated}
          <div class="truncation-alert">
            <Icon name="alert-triangle" size={14} />
            <span>Response truncated to 500KB (Original size: {(payload.full_size_bytes / 1024).toFixed(1)} KB)</span>
          </div>
        {/if}

        {#if viewMode === 'raw'}
          {#if payload?.response_text}
            <!-- IDE-Style Code Viewer with Line Gutter & Syntax Highlight -->
            <div class="code-viewer-wrap" bind:this={codeScrollContainer}>
              <pre class="code-gutter font-mono" aria-hidden="true">{gutterString}</pre>
              <pre class="code-body font-mono select-text">{@html highlightedContent.html}</pre>
            </div>
          {:else}
            <div class="empty-tab-content">
              <Icon name="file-text" size={32} />
              <span class="empty-title">Empty Response Body</span>
              <span class="empty-desc">The target server returned an empty body or headers only.</span>
            </div>
          {/if}
        {:else}
          <div class="preview-container">
            <iframe
              sandbox=""
              title="Sandboxed HTML Preview"
              srcdoc={previewSrcdoc}
              class="preview-iframe"
            ></iframe>
          </div>
        {/if}
      {:else if activeTab === 'request'}
        {#if payload?.request_text}
          <!-- Request View with Line Gutter -->
          <div class="code-viewer-wrap" bind:this={codeScrollContainer}>
            <pre class="code-gutter font-mono" aria-hidden="true">{gutterString}</pre>
            <pre class="code-body font-mono select-text">{@html highlightedContent.html}</pre>
          </div>
        {:else}
          <div class="empty-tab-content">
            <Icon name="terminal" size={32} />
            <span class="empty-title">No Raw Request Recorded</span>
            <span class="empty-desc">Raw request stream was not recorded for this result.</span>
          </div>
        {/if}
      {:else if activeTab === 'metadata'}
        <!-- Modern Structured Key-Value Metadata View -->
        <div class="metadata-view select-text">
          <div class="meta-card">
            <!-- Section 1: HTTP Summary -->
            <div class="meta-section">
              <div class="meta-section-header">
                <Icon name="globe" size={13} />
                <span>HTTP Response Summary</span>
              </div>
              <div class="meta-grid">
                <div class="meta-item">
                  <span class="meta-label">Status Code</span>
                  <div class="meta-value">
                    <span class="status-badge font-mono {getStatusBadgeClass(selectedItem.status)}">
                      <span class="badge-dot"></span>
                      {selectedItem.status} {getStatusReason(selectedItem.status)}
                    </span>
                  </div>
                </div>

                <div class="meta-item">
                  <span class="meta-label">Content Length</span>
                  <div class="meta-value font-mono">
                    <strong>{selectedItem.length.toLocaleString()}</strong> bytes
                    <span class="meta-sub">({formatBytes(selectedItem.length)})</span>
                  </div>
                </div>

                <div class="meta-item">
                  <span class="meta-label">Duration</span>
                  <div class="meta-value font-mono">
                    <span class="duration-badge">{selectedItem.duration_ms} ms</span>
                  </div>
                </div>

                <div class="meta-item">
                  <span class="meta-label">Content-Type</span>
                  <div class="meta-value font-mono">
                    <span class="type-pill">{payload?.content_type || selectedItem.content_type || 'Unknown'}</span>
                  </div>
                </div>

                {#if selectedItem.redirect_location}
                  <div class="meta-item full-width">
                    <span class="meta-label">Redirect Target</span>
                    <div class="meta-value font-mono redirect-val">
                      <Icon name="corner-down-right" size={12} />
                      <span class="selectable-link">{selectedItem.redirect_location}</span>
                    </div>
                  </div>
                {/if}
              </div>
            </div>

            <!-- Section 2: Engine Telemetry & Signatures -->
            <div class="meta-section">
              <div class="meta-section-header">
                <Icon name="database" size={13} />
                <span>Telemetry & Signatures</span>
              </div>
              <div class="meta-grid">
                <div class="meta-item">
                  <span class="meta-label">Result ID</span>
                  <div class="meta-value font-mono">
                    <span class="id-pill">#{selectedItem.id}</span>
                  </div>
                </div>

                <div class="meta-item">
                  <span class="meta-label">Words & Lines</span>
                  <div class="meta-value font-mono">
                    {selectedItem.words.toLocaleString()} words <span class="meta-sep">&bull;</span> {selectedItem.lines.toLocaleString()} lines
                  </div>
                </div>

                <div class="meta-item">
                  <span class="meta-label">Response Hash</span>
                  <div class="meta-value font-mono">
                    {#if selectedItem.hash}
                      <code class="hash-code">{selectedItem.hash}</code>
                    {:else}
                      <span class="meta-none">None</span>
                    {/if}
                  </div>
                </div>
              </div>
            </div>

            <!-- Section 3: Fuzz Parameters & Payloads -->
            {#if selectedItem.inputs && Object.keys(selectedItem.inputs).length > 0}
              <div class="meta-section">
                <div class="meta-section-header">
                  <Icon name="tag" size={13} />
                  <span>Fuzz Parameters & Payloads</span>
                </div>
                <div class="fuzz-inputs-list">
                  {#each Object.entries(selectedItem.inputs) as [param, value]}
                    <div class="fuzz-input-row font-mono">
                      <span class="fuzz-param-badge">{param}</span>
                      <Icon name="corner-down-right" size={12} />
                      <span class="fuzz-value">{value}</span>
                      <button
                        type="button"
                        class="btn-copy-inline"
                        onclick={() => copyToClipboard(value, param)}
                        title="Copy payload"
                      >
                        {#if copiedField === param}
                          <Icon name="check" size={12} />
                        {:else}
                          <Icon name="copy" size={12} />
                        {/if}
                      </button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}

            <!-- Section 4: Extracted Scraper Findings -->
            {#if selectedItem.scraper_data && Object.keys(selectedItem.scraper_data).length > 0}
              <div class="meta-section">
                <div class="meta-section-header">
                  <Icon name="search" size={13} />
                  <span>Extracted Scraper Findings</span>
                </div>
                <div class="fuzz-inputs-list">
                  {#each Object.entries(selectedItem.scraper_data) as [ruleName, extracted]}
                    <div class="fuzz-input-row font-mono">
                      <span class="fuzz-param-badge">{ruleName}</span>
                      <Icon name="corner-down-right" size={12} />
                      <span class="fuzz-value">{extracted}</span>
                      <button
                        type="button"
                        class="btn-copy-inline"
                        onclick={() => copyToClipboard(extracted, `scraper_${ruleName}`)}
                        title="Copy extracted finding"
                      >
                        {#if copiedField === `scraper_${ruleName}`}
                          <Icon name="check" size={12} />
                        {:else}
                          <Icon name="copy" size={12} />
                        {/if}
                      </button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .inspector-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    background-color: var(--surface);
    overflow: hidden;
    position: relative;
  }

  .empty-inspector {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    font-size: var(--text-sm);
    gap: 8px;
  }

  .empty-icon {
    font-size: 32px;
    opacity: 0.5;
  }

  /* 3-Column Grid guarantees tab-group is permanently centered with ZERO layout shift */
  .inspector-header {
    height: 38px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
    align-items: center;
    padding: 0 12px;
    box-sizing: border-box;
    font-size: var(--text-xs);
  }

  .header-left {
    justify-self: start;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    overflow: hidden;
  }

  .inspector-title {
    font-weight: var(--weight-bold);
    color: var(--text);
    font-size: var(--text-base);
    flex-shrink: 0;
  }

  .item-badge {
    background-color: var(--surface);
    border: 1px solid var(--border);
    padding: 1px 7px;
    border-radius: var(--radius-xs);
    color: var(--accent);
    font-weight: var(--weight-semibold);
    font-size: var(--text-xs);
    flex-shrink: 0;
  }

  .url-badge {
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    font-size: var(--text-xs);
  }

  /* Apple HIG Segmented Control Styling */
  .tab-group {
    justify-self: center;
    display: inline-flex;
    align-items: center;
    gap: 2px;
    background-color: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 2px;
    box-sizing: border-box;
  }

  .tab-btn {
    all: unset;
    height: 24px;
    padding: 0 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--text-secondary);
    transition: all 0.12s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
  }

  .tab-btn:hover:not(.active) {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .tab-btn.active {
    background-color: var(--surface);
    color: var(--text);
    font-weight: var(--weight-semibold);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
  }

  .header-right {
    justify-self: end;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .view-mode-toggle {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    background-color: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 2px;
    box-sizing: border-box;
  }

  .toggle-btn {
    all: unset;
    height: 22px;
    padding: 0 10px;
    border-radius: 3px;
    cursor: pointer;
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--text-secondary);
    transition: all 0.12s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
  }

  .toggle-btn:hover:not(.active) {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .toggle-btn.active {
    background-color: var(--surface);
    color: var(--text);
    font-weight: var(--weight-semibold);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .btn-beautify {
    all: unset;
    height: 26px;
    padding: 0 8px;
    border-radius: var(--radius-xs);
    border: 1px solid var(--border);
    background-color: var(--surface);
    cursor: pointer;
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--text);
    display: flex;
    align-items: center;
    gap: 5px;
    box-sizing: border-box;
    transition: all 0.12s ease;
  }

  .btn-beautify:hover {
    background-color: var(--surface-hover);
  }

  .btn-beautify.active {
    background-color: var(--bg-subtle);
    color: var(--accent);
    border-color: var(--accent);
    font-weight: var(--weight-semibold);
  }

  .btn-search-trigger {
    all: unset;
    width: 26px;
    height: 26px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-xs);
    border: 1px solid var(--border);
    background-color: var(--surface);
    cursor: pointer;
    color: var(--text-secondary);
    box-sizing: border-box;
    transition: all 0.12s ease;
  }

  .btn-search-trigger:hover,
  .btn-search-trigger.active {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .btn-search-trigger.active {
    color: var(--accent);
    border-color: var(--accent);
  }

  .btn-copy {
    all: unset;
    height: 26px;
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    padding: 0 10px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-xs);
    cursor: pointer;
    color: var(--text);
    display: flex;
    align-items: center;
    gap: 5px;
    box-sizing: border-box;
    transition: all 0.12s ease;
  }

  .btn-copy:hover {
    background-color: var(--surface-hover);
  }

  .btn-close {
    all: unset;
    width: 26px;
    height: 26px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-xs);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.12s ease;
  }

  .btn-close:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .inspector-content {
    flex: 1;
    overflow-y: hidden;
    position: relative;
    display: flex;
    flex-direction: column;
    background-color: var(--surface);
  }

  /* Floating Live Search Widget */
  .search-widget {
    position: absolute;
    top: 8px;
    right: 18px;
    z-index: 20;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.18), 0 1px 3px rgba(0, 0, 0, 0.1);
    backdrop-filter: blur(12px);
  }

  .search-input-wrap {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 2px 7px;
    background-color: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-xs);
    color: var(--text-secondary);
  }

  .search-input {
    all: unset;
    font-size: 11px;
    width: 150px;
    color: var(--text);
  }

  .match-badge {
    font-size: 10px;
    color: var(--text-secondary);
    min-width: 52px;
    text-align: center;
  }

  .match-badge.no-match {
    color: #ff453a;
    font-weight: var(--weight-bold);
  }

  .search-nav-btns {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .search-nav-btn {
    all: unset;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
    cursor: pointer;
    color: var(--text);
    transition: background 0.1s ease;
  }

  .search-nav-btn:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .search-nav-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .search-close-btn {
    all: unset;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
    cursor: pointer;
    color: var(--text-secondary);
    transition: all 0.1s ease;
  }

  .search-close-btn:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .truncation-alert {
    background-color: rgba(255, 149, 0, 0.12);
    color: var(--status-4xx);
    padding: 8px 14px;
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    border-bottom: 1px solid rgba(255, 149, 0, 0.25);
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  /* IDE-Style Code Viewer Container */
  .code-viewer-wrap {
    display: flex;
    flex: 1;
    overflow: auto;
    background-color: var(--surface);
    position: relative;
    box-sizing: border-box;
  }

  /* Sticky Gutter with Line Numbers */
  .code-gutter {
    margin: 0;
    padding: 14px 10px 14px 12px;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 20px;
    text-align: right;
    color: var(--text-secondary);
    opacity: 0.45;
    user-select: none;
    border-right: 1px solid var(--border);
    background-color: var(--bg-subtle);
    min-width: 44px;
    box-sizing: border-box;
    white-space: pre;
    position: sticky;
    left: 0;
    z-index: 2;
  }

  /* Code Body with Horizontal Scroll & Tab Alignment */
  .code-body {
    margin: 0;
    padding: 14px 18px;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 20px;
    color: var(--text);
    white-space: pre;
    flex: 1;
    tab-size: 2;
    user-select: text;
    box-sizing: border-box;
  }

  /* Syntax Highlighting Tokens (Bound directly to centralized Theme Tokens) */
  :global(.tok-tag) {
    color: var(--code-tag);
    font-weight: 600;
  }
  :global(.tok-attr) {
    color: var(--code-attr);
  }
  :global(.tok-str) {
    color: var(--code-str);
  }
  :global(.tok-key) {
    color: var(--code-key);
    font-weight: 600;
  }
  :global(.tok-num) {
    color: var(--code-num);
  }
  :global(.tok-bool) {
    color: var(--code-bool);
    font-weight: 600;
  }
  :global(.tok-comm) {
    color: var(--code-comm);
    font-style: italic;
  }
  :global(.tok-doctype) {
    color: var(--code-doctype);
    font-weight: 600;
  }
  :global(.tok-ent) {
    color: var(--code-ent);
  }
  :global(.tok-prop) {
    color: var(--code-prop);
  }
  :global(.tok-val) {
    color: var(--code-val);
  }
  :global(.tok-sel) {
    color: var(--code-sel, var(--code-ent));
    font-weight: 600;
  }

  /* Search Highlights */
  :global(.search-match) {
    background-color: rgba(255, 214, 10, 0.35);
    color: inherit;
    border-radius: 2px;
    padding: 0 1px;
    outline: 1px solid rgba(255, 214, 10, 0.55);
  }

  :global(.search-match.active-match) {
    background-color: #ff9f0a;
    color: #000000;
    font-weight: 700;
    outline: 2px solid #ff9f0a;
    box-shadow: 0 0 10px rgba(255, 159, 10, 0.7);
  }

  .empty-tab-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
    color: var(--text-secondary);
    padding: 36px 16px;
  }

  .empty-title {
    font-size: var(--text-base);
    font-weight: var(--weight-semibold);
    color: var(--text);
  }

  .empty-desc {
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .preview-container {
    flex: 1;
    width: 100%;
    height: 100%;
    position: relative;
    background: #ffffff;
  }

  .preview-iframe {
    width: 100%;
    height: 100%;
    border: none;
    background: #ffffff;
  }

  /* Metadata Structured Cards */
  .metadata-view {
    padding: 16px;
    overflow-y: auto;
    flex: 1;
    box-sizing: border-box;
  }

  .meta-card {
    max-width: 860px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .meta-section {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  }

  .meta-section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 14px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    font-size: var(--text-xs);
    font-weight: var(--weight-semibold);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .meta-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 1px;
    background-color: var(--border);
  }

  .meta-item {
    background-color: var(--surface);
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta-item.full-width {
    grid-column: 1 / -1;
  }

  .meta-label {
    font-size: var(--text-xs);
    color: var(--text-secondary);
    font-weight: var(--weight-medium);
  }

  .meta-value {
    font-size: var(--text-sm);
    color: var(--text);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .meta-sub {
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .meta-sep {
    color: var(--text-secondary);
    opacity: 0.6;
    margin: 0 2px;
  }

  .meta-none {
    color: var(--text-secondary);
    font-style: italic;
    opacity: 0.7;
  }

  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 2px 8px;
    border-radius: var(--radius-xs);
    font-size: var(--text-xs);
    font-weight: var(--weight-semibold);
    line-height: 16px;
  }

  .badge-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: currentColor;
  }

  .badge-2xx {
    background-color: rgba(52, 199, 89, 0.14);
    color: var(--status-2xx);
    border: 1px solid rgba(52, 199, 89, 0.3);
  }

  .badge-3xx {
    background-color: rgba(0, 122, 255, 0.14);
    color: var(--status-3xx);
    border: 1px solid rgba(0, 122, 255, 0.3);
  }

  .badge-4xx {
    background-color: rgba(255, 149, 0, 0.14);
    color: var(--status-4xx);
    border: 1px solid rgba(255, 149, 0, 0.3);
  }

  .badge-5xx {
    background-color: rgba(255, 59, 48, 0.14);
    color: var(--status-5xx);
    border: 1px solid rgba(255, 59, 48, 0.3);
  }

  .badge-err {
    background-color: rgba(142, 142, 147, 0.14);
    color: var(--status-err);
    border: 1px solid rgba(142, 147, 0.3);
  }

  .id-pill {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 7px;
    color: var(--accent);
    font-weight: var(--weight-bold);
    font-size: var(--text-xs);
  }

  .type-pill {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 2px 8px;
    font-size: var(--text-xs);
    color: var(--text);
  }

  .hash-code {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    padding: 2px 7px;
    border-radius: 4px;
    font-size: 11px;
    color: var(--text);
  }

  .duration-badge {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    padding: 2px 8px;
    border-radius: 4px;
    font-size: var(--text-xs);
  }

  .redirect-val {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--accent);
    word-break: break-all;
  }

  .selectable-link {
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .fuzz-inputs-list {
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .fuzz-input-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
  }

  .fuzz-param-badge {
    background-color: var(--accent);
    color: var(--accent-contrast);
    font-weight: 700;
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 3px;
    letter-spacing: 0.5px;
  }

  .fuzz-value {
    color: var(--text);
    font-weight: 500;
    flex: 1;
    word-break: break-all;
  }

  .btn-copy-inline {
    all: unset;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 4px;
    border-radius: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.12s ease;
  }

  .btn-copy-inline:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .loading-state,
  .error-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .error-state {
    color: #ff3b30;
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
