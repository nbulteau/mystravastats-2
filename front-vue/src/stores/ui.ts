import { defineStore } from "pinia";
import type { Toast } from "@/models/toast.model";

export const useUiStore = defineStore("ui", {
  state: () => ({
    toasts: [] as Toast[],
  }),
  actions: {
    showToast(toast: Toast) {
      this.toasts.push(toast);
      if (toast.timeout) {
        setTimeout(() => {
          this.removeToast(toast);
        }, toast.timeout);
      }
    },
    removeToast(toast: Toast) {
      this.toasts = this.toasts.filter((item) => item.id !== toast.id);
    },
  },
});
