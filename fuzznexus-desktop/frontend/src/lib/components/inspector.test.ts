import { describe, it, expect } from 'vitest';

describe('Inspector & Footer Formatting Helpers', () => {
  function getStatusCategory(status: number): '2xx' | '3xx' | '4xx' | '5xx' | 'err' {
    if (status >= 200 && status < 300) return '2xx';
    if (status >= 300 && status < 400) return '3xx';
    if (status >= 400 && status < 500) return '4xx';
    if (status >= 500 && status < 600) return '5xx';
    return 'err';
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

  function detectLanguage(contentType: string, text: string): 'json' | 'html' | 'xml' | 'text' {
    const ct = (contentType || '').toLowerCase();
    if (ct.includes('application/json') || ct.includes('+json') || ct.includes('/json')) return 'json';
    if (ct.includes('text/html') || ct.includes('application/xhtml')) return 'html';
    if (ct.includes('xml')) return 'xml';

    const trimmed = text.trim();
    if (!trimmed) return 'text';

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

  function beautifyContent(text: string, lang: 'json' | 'html' | 'xml' | 'text'): string {
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

  function applySearchHighlight(html: string, query: string, activeIdx: number): { html: string; count: number } {
    if (!query || !query.trim()) {
      return { html, count: 0 };
    }
    const escapedQuery = escapeHTML(query.trim());
    const regexQuery = escapeRegex(escapedQuery);
    const re = new RegExp(`(${regexQuery})(?![^<]*>)`, 'gi');
    let matchCounter = 0;

    const res = html.replace(re, (m) => {
      const idx = matchCounter++;
      const isActive = idx === activeIdx;
      return `<mark id="search-match-${idx}" class="search-match ${isActive ? 'active-match' : ''}">${m}</mark>`;
    });

    return { html: res, count: matchCounter };
  }

  it('correctly maps HTTP status codes to categories', () => {
    expect(getStatusCategory(200)).toBe('2xx');
    expect(getStatusCategory(201)).toBe('2xx');
    expect(getStatusCategory(301)).toBe('3xx');
    expect(getStatusCategory(302)).toBe('3xx');
    expect(getStatusCategory(403)).toBe('4xx');
    expect(getStatusCategory(404)).toBe('4xx');
    expect(getStatusCategory(500)).toBe('5xx');
    expect(getStatusCategory(502)).toBe('5xx');
    expect(getStatusCategory(0)).toBe('err');
  });

  it('formats byte lengths into human-readable strings', () => {
    expect(formatBytes(500)).toBe('500 B');
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1702)).toBe('1.7 KB');
    expect(formatBytes(1048576)).toBe('1.00 MB');
    expect(formatBytes(3500000)).toBe('3.34 MB');
  });

  it('resolves standard HTTP status phrases', () => {
    expect(getStatusReason(200)).toBe('OK');
    expect(getStatusReason(404)).toBe('Not Found');
    expect(getStatusReason(500)).toBe('Internal Server Error');
    expect(getStatusReason(999)).toBe('');
  });

  it('detects language from Content-Type and payload body', () => {
    expect(detectLanguage('application/json', '{"status": 200}')).toBe('json');
    expect(detectLanguage('text/html; charset=utf-8', '<html></html>')).toBe('html');
    expect(detectLanguage('', '{"auto": "detected"}')).toBe('json');
    expect(detectLanguage('', '<html><body>test</body></html>')).toBe('html');
    expect(detectLanguage('', '<!doctype html><html><body>modern</body></html>')).toBe('html');
    expect(detectLanguage('', '<div><span>partial fragment</span></div>')).toBe('html');
    expect(detectLanguage('text/plain', 'simple raw text')).toBe('text');
  });

  it('beautifies minified JSON payloads accurately', () => {
    const raw = '{"a":1,"b":["item1","item2"]}';
    const formatted = beautifyContent(raw, 'json');
    expect(formatted).toBe('{\n  "a": 1,\n  "b": [\n    "item1",\n    "item2"\n  ]\n}');
  });

  it('beautifies minified HTML and deeply formats CSS rules inside <style>', () => {
    const raw = '<html><head><style>body { background: #dedede; color: #fff; } div.navbar { top: 10px; }</style></head><body><h1>Hi</h1></body></html>';
    const formatted = beautifyContent(raw, 'html');
    // Must contain indented CSS declarations
    expect(formatted).toContain('background: #dedede;');
    expect(formatted).toContain('color: #fff;');
    expect(formatted).toContain('top: 10px;');
    expect(formatted).toContain('<h1>\n      Hi\n    </h1>');
  });

  it('safely applies search highlights avoiding HTML tags', () => {
    const rawHTML = '<span class="tok-tag">&lt;div</span> <span class="tok-attr">class</span>';
    const result = applySearchHighlight(rawHTML, 'class', 0);
    expect(result.count).toBe(1);
    expect(result.html).toContain('<mark id="search-match-0" class="search-match active-match">class</mark>');
    expect(result.html).toContain('class="tok-tag"');
  });

  it('safely escapes special regex characters in search query without crashing', () => {
    const rawHTML = '<span>Checking [test]*?(token) here</span>';
    const result = applySearchHighlight(rawHTML, '[test]*?(token)', 0);
    expect(result.count).toBe(1);
    expect(result.html).toContain('<mark id="search-match-0" class="search-match active-match">[test]*?(token)</mark>');
  });

  it('correctly formats and highlights CSS selectors and properties', () => {
    const cssFragment = 'body {\n  background: #dedede;\n}';
    let highlighted = cssFragment
      .replace(/^(\s*)([a-zA-Z-]+)(\s*:\s*)([^;<>{}\n]+)(;?)/gm, (_, space, prop, colon, val, semi) => {
        return `${space}<span class="tok-prop">${prop}</span>${colon}<span class="tok-val">${val}</span>${semi}`;
      })
      .replace(/^(\s*)([.#a-zA-Z0-9_:\s,-]+)(\s*\{)$/gm, (m, space, sel, brace) => {
        if (sel.includes('&lt;') || sel.includes('&gt;')) return m;
        return `${space}<span class="tok-sel">${sel}</span>${brace}`;
      });

    expect(highlighted).toContain('<span class="tok-sel">body </span>{');
    expect(highlighted).toContain('<span class="tok-prop">background</span>: <span class="tok-val">#dedede</span>;');
  });

  it('accurately highlights HTML tags and attributes without DOM corruption', () => {
    const rawHTML = '<!DOCTYPE html><div class="container" id="main"><p>Hello &copy; welcome</p><input type="text" disabled /></div>';
    const highlighted = highlightTokens(rawHTML, 'html');

    // DOCTYPE
    expect(highlighted).toContain('<span class="tok-doctype">&lt;!DOCTYPE html&gt;</span>');
    // Tag names
    expect(highlighted).toContain('&lt;<span class="tok-tag">div</span>');
    expect(highlighted).toContain('&lt;/<span class="tok-tag">div</span>&gt;');
    expect(highlighted).toContain('&lt;<span class="tok-tag">p</span>&gt;');
    expect(highlighted).toContain('&lt;<span class="tok-tag">input</span>');
    // Attributes
    expect(highlighted).toContain('<span class="tok-attr">class</span>=<span class="tok-str">&quot;container&quot;</span>');
    expect(highlighted).toContain('<span class="tok-attr">id</span>=<span class="tok-str">&quot;main&quot;</span>');
    expect(highlighted).toContain('<span class="tok-attr">type</span>=<span class="tok-str">&quot;text&quot;</span>');
    expect(highlighted).toContain('<span class="tok-attr">disabled</span>');
    // Entities
    expect(highlighted).toContain('<span class="tok-ent">&amp;copy;</span>');
    // Ensure no malformed tags (<span <span...)
    expect(highlighted).not.toContain('<span <span');
    expect(highlighted).not.toContain('class="tok-tag"&gt;div');
  });

  it('accurately highlights JSON keys, values, numbers and booleans without token collision', () => {
    const rawJSON = '{\n  "status": 200,\n  "msg": "user 123 active: true",\n  "active": true,\n  "ratio": -0.45\n}';
    const highlighted = highlightTokens(rawJSON, 'json');

    // Keys
    expect(highlighted).toContain('<span class="tok-key">&quot;status&quot;</span>:');
    expect(highlighted).toContain('<span class="tok-key">&quot;msg&quot;</span>:');
    expect(highlighted).toContain('<span class="tok-key">&quot;active&quot;</span>:');
    expect(highlighted).toContain('<span class="tok-key">&quot;ratio&quot;</span>:');
    // String value (numbers/booleans inside string literal must NOT be touched)
    expect(highlighted).toContain('<span class="tok-str">&quot;user 123 active: true&quot;</span>');
    expect(highlighted).not.toContain('user <span class="tok-num">123</span>');
    // Numbers
    expect(highlighted).toContain('<span class="tok-num">200</span>');
    expect(highlighted).toContain('<span class="tok-num">-0.45</span>');
    // Booleans
    expect(highlighted).toContain('<span class="tok-bool">true</span>');
  });

  it('accurately highlights HTTP raw request lines, headers and body', () => {
    const rawReq = 'GET /?internal=true HTTP/1.1\r\nHost: sabzlearn.ir\r\nUser-Agent: ffuf/2.1\r\n\r\n{"username":"admin"}';
    const highlighted = highlightTokens(rawReq, 'http');

    // Method, Path, Protocol
    expect(highlighted).toContain('<span class="tok-tag">GET</span> <span class="tok-attr">/?internal=true</span> <span class="tok-doctype">HTTP/1.1</span>');
    // Headers
    expect(highlighted).toContain('<span class="tok-prop">Host</span>: <span class="tok-val">sabzlearn.ir</span>');
    expect(highlighted).toContain('<span class="tok-prop">User-Agent</span>: <span class="tok-val">ffuf/2.1</span>');
    // Body formatted as JSON
    expect(highlighted).toContain('<span class="tok-key">&quot;username&quot;</span>:<span class="tok-str">&quot;admin&quot;</span>');
  });

  it('accurately highlights responses larger than 300KB (e.g. 455KB response)', () => {
    const bigHTML = '<html><body>' + '<p class="content">Paragraph text</p>\n'.repeat(12000) + '</body></html>';
    expect(bigHTML.length).toBeGreaterThan(400000); // ~450KB
    const highlighted = highlightTokens(bigHTML, 'html');
    expect(highlighted).toContain('<span class="tok-tag">html</span>');
    expect(highlighted).toContain('<span class="tok-attr">class</span>=<span class="tok-str">&quot;content&quot;</span>');
  });

  it('validates reset and clear workflow state contracts', () => {
    // Model state before reset
    const scanState = {
      sessionID: 42,
      state: 'Completed',
      lastError: 'some error',
      reqCount: 100,
      reqTotal: 100,
      reqSec: 50,
      errorCount: 2,
      percentage: 100
    };

    // Simulate reset()
    function resetScan(s: typeof scanState) {
      if (s.state === 'Running') return;
      s.sessionID = 0;
      s.state = 'Idle';
      s.lastError = '';
      s.reqCount = 0;
      s.reqTotal = 0;
      s.reqSec = 0;
      s.errorCount = 0;
      s.percentage = 0;
    }

    resetScan(scanState);
    expect(scanState.sessionID).toBe(0);
    expect(scanState.state).toBe('Idle');
    expect(scanState.reqTotal).toBe(0);
    expect(scanState.percentage).toBe(0);
    expect(scanState.lastError).toBe('');
  });
});
