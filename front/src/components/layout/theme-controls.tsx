import { Moon, Palette, SunMedium } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme, type AccentTheme } from "@/app/theme-provider";

const accentOptions: Array<{ value: AccentTheme; label: string }> = [
  { value: "amber", label: "Amber" },
  { value: "teal", label: "Teal" },
  { value: "ocean", label: "Ocean" },
];

export function ThemeControls() {
  return <ThemeControlsPanel />;
}

export function ThemeControlsPanel({ inset = false }: { inset?: boolean }) {
  const { mode, accent, setAccent, toggleMode } = useTheme();

  return (
    <div
      className={cn(
        "rounded-3xl border border-border/70 bg-card/85 p-4 shadow-panel backdrop-blur",
        inset && "border-0 bg-transparent p-0 shadow-none backdrop-blur-0",
      )}
    >
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">Appearance</p>
            <p className="mt-1 text-sm text-foreground">Choose a theme color and toggle dark mode.</p>
          </div>
          <button
            type="button"
            onClick={toggleMode}
            className="inline-flex h-11 items-center gap-2 rounded-full border border-border bg-background px-4 text-sm font-medium text-foreground transition hover:bg-accent hover:text-accent-foreground"
            aria-label={mode === "dark" ? "Switch to light mode" : "Switch to dark mode"}
          >
            {mode === "dark" ? <SunMedium className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            {mode === "dark" ? "Light" : "Dark"}
          </button>
        </div>

        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs uppercase tracking-[0.22em] text-muted-foreground">
            <Palette className="h-3.5 w-3.5" />
            Accent
          </div>
          <div className="grid grid-cols-3 gap-2">
            {accentOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => setAccent(option.value)}
                className={cn(
                  "rounded-2xl border px-3 py-3 text-sm font-medium transition",
                  accent === option.value
                    ? "border-foreground bg-foreground text-background"
                    : "border-border bg-background text-foreground hover:bg-secondary",
                )}
                aria-pressed={accent === option.value}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
