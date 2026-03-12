import { create } from "zustand";
import { persist } from "zustand/middleware";

// --- Types ---
export type Theme = "light" | "dark";

const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
  ? "dark"
  : "light";

interface AppStore {
  // Theme
  theme: Theme;
  setTheme: (theme: Theme) => void;

  User: string;
}

export const useAppStore = create<AppStore>()(
  persist(
    (set) => ({
      // Theme
      theme: systemTheme as Theme,
      setTheme: (theme) => set({ theme }),
      User: "name",
    }),
    {
      name: "app-store",
      partialize: (s): Partial<AppStore> => ({ theme: s.theme }),
    },
  ),
);
