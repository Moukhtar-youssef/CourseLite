import { useEffect } from "react";
import { useAppStore } from "./store";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const { theme } = useAppStore();

  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove("light", "dark");

    root.classList.add(theme);
  }, [theme]);

  return <>{children}</>;
}
