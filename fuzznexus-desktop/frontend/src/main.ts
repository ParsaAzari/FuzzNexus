import './style.css'
import App from './App.svelte'
import { mount } from 'svelte'
import { Environment } from '../wailsjs/runtime/runtime'

// Detect OS platform and set data-platform attribute on root element
async function initPlatform() {
  try {
    if (typeof window !== 'undefined' && (window as any).runtime?.Environment) {
      const env = await Environment()
      if (env?.platform) {
        document.documentElement.setAttribute('data-platform', env.platform)
        return
      }
    }
  } catch {
    // ignore runtime fetch errors in headless test environments
  }

  // Fallback heuristic based on navigator
  if (typeof navigator !== 'undefined') {
    const ua = navigator.userAgent.toLowerCase()
    const plat = (navigator.platform || '').toLowerCase()
    if (ua.includes('mac') || plat.includes('mac')) {
      document.documentElement.setAttribute('data-platform', 'darwin')
    } else if (ua.includes('win') || plat.includes('win')) {
      document.documentElement.setAttribute('data-platform', 'windows')
    } else {
      document.documentElement.setAttribute('data-platform', 'linux')
    }
  }
}

initPlatform()

const app = mount(App, {
  target: document.getElementById('app')!
})

export default app
