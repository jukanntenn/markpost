import "@testing-library/jest-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ThemeToggle from "./ThemeToggle";
import { ThemeProvider } from "../contexts/ThemeProvider";
import { renderWithI18n } from "../test/utils";

function renderWithTheme() {
  return renderWithI18n(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>
  );
}

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe("ThemeToggle", () => {
  it("renders theme toggle button", () => {
    renderWithTheme();
    const toggle = screen.getByRole("button", { name: /theme/i });
    expect(toggle).toBeInTheDocument();
  });

  it("shows system icon by default", () => {
    renderWithTheme();
    const toggle = screen.getByRole("button", { name: /theme/i });
    const svg = toggle.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg).toHaveClass("bi-circle-half");
  });

  it("opens dropdown menu on click", async () => {
    const user = userEvent.setup();
    renderWithTheme();

    const toggle = screen.getByRole("button", { name: /theme/i });
    await user.click(toggle);

    expect(screen.getByText(/light/i)).toBeInTheDocument();
    expect(screen.getByText(/dark/i)).toBeInTheDocument();
    expect(screen.getByText(/system/i)).toBeInTheDocument();
  });

  it("switches to light theme when light is clicked", async () => {
    const user = userEvent.setup();
    renderWithTheme();

    const toggle = screen.getByRole("button", { name: /theme/i });
    await user.click(toggle);

    const lightOption = screen.getByText(/light/i);
    await user.click(lightOption);

    expect(localStorage.getItem("theme-mode")).toBe("light");
  });

  it("switches to dark theme when dark is clicked", async () => {
    const user = userEvent.setup();
    renderWithTheme();

    const toggle = screen.getByRole("button", { name: /theme/i });
    await user.click(toggle);

    const darkOption = screen.getByText(/dark/i);
    await user.click(darkOption);

    expect(localStorage.getItem("theme-mode")).toBe("dark");
  });

  it("marks active theme in dropdown", async () => {
    const user = userEvent.setup();
    localStorage.setItem("theme-mode", "light");
    renderWithTheme();

    const toggle = screen.getByRole("button", { name: /theme/i });
    await user.click(toggle);

    const lightItem = screen.getByText(/light/i).closest(".dropdown-item");
    const darkItem = screen.getByText(/dark/i).closest(".dropdown-item");
    const systemItem = screen.getByText(/system/i).closest(".dropdown-item");

    expect(lightItem).toHaveClass("active");
    expect(darkItem).not.toHaveClass("active");
    expect(systemItem).not.toHaveClass("active");
  });
});
