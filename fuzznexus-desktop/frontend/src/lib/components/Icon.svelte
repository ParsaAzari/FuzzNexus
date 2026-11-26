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
    ChevronUp,
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
    Minus,
    Trash2,
    Edit3,
    Save,
    Download,
    Upload,
    RefreshCw,
    ShieldAlert,
    ShieldCheck,
    RotateCcw,
    Ban,
    Info,
    Puzzle,
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
    PanelLeftOpen,
    Lock,
    LockOpen,
    Tag,
    Target,
    ExternalLink,
    Braces,
    Code,
    CodeXml,
    Palette
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
    puzzle: Puzzle,

    // Navigation & Sort
    'arrow-up': ArrowUp,
    'arrow-down': ArrowDown,
    'arrow-up-down': ArrowUpDown,
    'chevron-up': ChevronUp,
    'chevron-down': ChevronDown,
    'chevron-right': ChevronRight,
    'corner-down-right': CornerDownRight,

    // UI Tools & Chrome
    search: Search,
    code: Code,
    braces: Braces,
    'code-xml': CodeXml,
    x: X,
    close: X,
    copy: Copy,
    filter: Filter,
    sliders: SlidersHorizontal,
    settings: Settings,
    plus: Plus,
    minus: Minus,
    trash: Trash2,
    'trash-2': Trash2,
    edit: Edit3,
    'edit-2': Edit3,
    'edit-3': Edit3,
    save: Save,
    download: Download,
    upload: Upload,
    refresh: RefreshCw,
    'rotate-ccw': RotateCcw,
    'shield-check': ShieldCheck,
    'shield-alert': ShieldAlert,
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
    lock: Lock,
    unlock: LockOpen,
    tag: Tag,
    target: Target,
    'external-link': ExternalLink,

    // Themes
    palette: Palette,
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
