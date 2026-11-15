<script lang="ts">
  import type { Component } from 'svelte';
  import {
    Play,
    Pause,
    Square,
    Zap,
    AlertTriangle,
    AlertCircle,
    AlertOctagon,
    ArrowUp,
    ArrowDown,
    ArrowUpDown,
    ChevronDown,
    ChevronRight,
    CornerDownRight,
    Search,
    X,
    Check,
    CheckCircle2,
    Copy,
    Sun,
    Moon,
    Laptop,
    Monitor,
    FileText,
    Layers,
    Filter,
    Plus,
    Trash2,
    Edit3,
    Save,
    Download,
    Upload,
    RefreshCw,
    ShieldAlert,
    Ban,
    Info,
    Terminal,
    Settings,
    Globe,
    Database,
    SlidersHorizontal,
    Undo2,
    Redo2,
    FolderOpen,
    Eye,
    EyeOff,
    Ellipsis,
    PanelLeftClose,
    PanelLeftOpen
  } from '@lucide/svelte';

  const iconMap: Record<string, Component<any>> = {
    // Actions & Engine Controls
    play: Play,
    pause: Pause,
    square: Square,
    stop: Square,
    zap: Zap,

    // Status & Feedback
    check: Check,
    'check-circle': CheckCircle2,
    'status-2xx': CheckCircle2,
    'status-3xx': CornerDownRight,
    'status-4xx': ShieldAlert,
    'status-5xx': AlertOctagon,
    'status-err': Ban,
    'alert-triangle': AlertTriangle,
    'alert-circle': AlertCircle,
    'alert-octagon': AlertOctagon,
    info: Info,

    // Navigation & Sort
    'arrow-up': ArrowUp,
    'arrow-down': ArrowDown,
    'arrow-up-down': ArrowUpDown,
    'chevron-down': ChevronDown,
    'chevron-right': ChevronRight,
    'corner-down-right': CornerDownRight,

    // UI Tools & Chrome
    search: Search,
    x: X,
    close: X,
    copy: Copy,
    filter: Filter,
    sliders: SlidersHorizontal,
    settings: Settings,
    plus: Plus,
    trash: Trash2,
    edit: Edit3,
    save: Save,
    download: Download,
    upload: Upload,
    refresh: RefreshCw,
    undo: Undo2,
    redo: Redo2,
    folder: FolderOpen,
    eye: Eye,
    'eye-off': EyeOff,
    ellipsis: Ellipsis,
    more: Ellipsis,
    'panel-left': PanelLeftClose,
    'panel-left-close': PanelLeftClose,
    'panel-left-open': PanelLeftOpen,

    // Themes
    sun: Sun,
    moon: Moon,
    laptop: Laptop,
    monitor: Monitor,

    // Data & Objects
    'file-text': FileText,
    layers: Layers,
    terminal: Terminal,
    globe: Globe,
    database: Database
  };

  interface Props {
    name: string;
    size?: number | string;
    strokeWidth?: number;
    color?: string;
    class?: string;
    'aria-hidden'?: boolean | 'true' | 'false';
  }

  let {
    name,
    size = 16,
    strokeWidth = 2,
    color = 'currentColor',
    class: className = '',
    'aria-hidden': ariaHidden = true
  }: Props = $props();

  let IconComponent = $derived(iconMap[name.toLowerCase()] || Info);
</script>

<span class="inline-icon {className}" aria-hidden={ariaHidden}>
  <IconComponent {size} {strokeWidth} {color} />
</span>

<style>
  .inline-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    line-height: 1;
    flex-shrink: 0;
  }
</style>
