export interface Toast {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  message: string;
  duration?: number;
}

class ToastStore {
  toasts = $state<Toast[]>([]);

  addToast(message: string, type: 'info' | 'success' | 'warning' | 'error' = 'info', duration: number = 4000) {
    const id = `toast_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`;
    const newToast: Toast = { id, type, message, duration };
    this.toasts = [...this.toasts, newToast];

    if (duration > 0) {
      setTimeout(() => {
        this.removeToast(id);
      }, duration);
    }
    return id;
  }

  success(message: string, duration?: number) {
    return this.addToast(message, 'success', duration);
  }

  error(message: string, duration: number = 6000) {
    return this.addToast(message, 'error', duration);
  }

  warning(message: string, duration?: number) {
    return this.addToast(message, 'warning', duration);
  }

  info(message: string, duration?: number) {
    return this.addToast(message, 'info', duration);
  }

  removeToast(id: string) {
    this.toasts = this.toasts.filter((t) => t.id !== id);
  }

  clear() {
    this.toasts = [];
  }
}

export const toastStore = new ToastStore();
