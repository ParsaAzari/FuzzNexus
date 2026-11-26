import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig(({ mode }) => {
  const isDev = mode === 'development';
  const csp = isDev
    ? "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' wails: http://wails.localhost ws://localhost:* ipc:; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none';"
    : "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' wails: http://wails.localhost; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none';";

  return {
    plugins: [
      svelte(),
      {
        name: 'html-transform',
        transformIndexHtml(html) {
          return html.replace('%CSP_DIRECTIVE%', csp);
        }
      }
    ],
    build: {
      sourcemap: false,
      minify: 'esbuild'
    },
    esbuild: {
      legalComments: 'none',
      drop: isDev ? [] : ['console', 'debugger']
    }
  };
});
