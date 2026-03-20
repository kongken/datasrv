import {
  createContext,
  useContext,
  useEffect,
  useState,
  type PropsWithChildren,
} from "react";

type ThemeMode = "light" | "dark";
type AccentTheme = "amber" | "teal" | "ocean";

type ThemeContextValue = {
  mode: ThemeMode;
  accent: AccentTheme;
  setMode: (mode: ThemeMode) => void;
  setAccent: (accent: AccentTheme) => void;
  toggleMode: () => void;
};

const THEME_MODE_STORAGE_KEY = "betahub-theme-mode";
const THEME_ACCENT_STORAGE_KEY = "betahub-theme-accent";

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getPreferredMode(): ThemeMode {
  if (typeof window === "undefined") {
    return "light";
  }

  const storedMode = window.localStorage.getItem(THEME_MODE_STORAGE_KEY);
  if (storedMode === "light" || storedMode === "dark") {
    return storedMode;
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function getPreferredAccent(): AccentTheme {
  if (typeof window === "undefined") {
    return "amber";
  }

  const storedAccent = window.localStorage.getItem(THEME_ACCENT_STORAGE_KEY);
  if (storedAccent === "amber" || storedAccent === "teal" || storedAccent === "ocean") {
    return storedAccent;
  }

  return "amber";
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const [mode, setMode] = useState<ThemeMode>("light");
  const [accent, setAccent] = useState<AccentTheme>("amber");

  useEffect(() => {
    setMode(getPreferredMode());
    setAccent(getPreferredAccent());
  }, []);

  useEffect(() => {
    if (typeof document === "undefined") {
      return;
    }

    const root = document.documentElement;
    root.dataset.theme = mode;
    root.dataset.accent = accent;
    window.localStorage.setItem(THEME_MODE_STORAGE_KEY, mode);
    window.localStorage.setItem(THEME_ACCENT_STORAGE_KEY, accent);
  }, [mode, accent]);

  return (
    <ThemeContext.Provider
      value={{
        mode,
        accent,
        setMode,
        setAccent,
        toggleMode: () => setMode((currentMode) => (currentMode === "light" ? "dark" : "light")),
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }

  return context;
}

export type { AccentTheme, ThemeMode };
