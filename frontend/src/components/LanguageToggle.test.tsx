import "@testing-library/jest-dom";

import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";

import { I18nextProvider } from "react-i18next";
import LanguageToggle from "./LanguageToggle";
import i18n from "../i18n";
import userEvent from "@testing-library/user-event";

function renderWithI18n() {
  return render(
    <I18nextProvider i18n={i18n}>
      <LanguageToggle />
    </I18nextProvider>
  );
}

function findDropdownItemByText(text: string) {
  const elements = screen.getAllByText(text);
  for (const el of elements) {
    const item = el.closest(".dropdown-item");
    if (item) return item as HTMLElement;
  }
  return null;
}

beforeEach(() => {
  localStorage.clear();
});

describe("LanguageToggle", () => {
  it("renders English label on initial English language", async () => {
    await i18n.changeLanguage("en");
    renderWithI18n();
    const toggle = screen.getByRole("button", {
      name: i18n.t("language.changeLanguage"),
    });
    expect(toggle).toBeInTheDocument();
    expect(screen.getByText("English")).toBeInTheDocument();
  });

  it("switches to Chinese after selecting Chinese", async () => {
    await i18n.changeLanguage("en");
    renderWithI18n();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: i18n.t("language.changeLanguage") })
    );
    await user.click(findDropdownItemByText("中文")!);
    expect(i18n.language).toBe("zh");
    const toggle = screen.getByRole("button", {
      name: i18n.t("language.changeLanguage"),
    });
    expect(within(toggle).getByText("中文")).toBeInTheDocument();
  });

  it("switches to English after selecting English", async () => {
    await i18n.changeLanguage("zh");
    renderWithI18n();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: i18n.t("language.changeLanguage") })
    );
    await user.click(findDropdownItemByText("English")!);
    expect(i18n.language).toBe("en");
    const toggle = screen.getByRole("button", {
      name: i18n.t("language.changeLanguage"),
    });
    expect(within(toggle).getByText("English")).toBeInTheDocument();
  });

  it("invokes changeLanguage when selecting menu item", async () => {
    await i18n.changeLanguage("en");
    const spy = vi.spyOn(i18n, "changeLanguage");
    renderWithI18n();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: i18n.t("language.changeLanguage") })
    );
    await user.click(findDropdownItemByText("中文")!);
    expect(spy).toHaveBeenCalledWith("zh");
    spy.mockRestore();
  });

  it("marks English item active when language is English", async () => {
    await i18n.changeLanguage("en");
    renderWithI18n();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: i18n.t("language.changeLanguage") })
    );
    const englishItem = findDropdownItemByText("English");
    const chineseItem = findDropdownItemByText("中文");
    expect(englishItem).not.toBeNull();
    expect(chineseItem).not.toBeNull();
    expect(englishItem!).toHaveClass("active");
    expect(chineseItem!).not.toHaveClass("active");
  });

  it("marks Chinese item active when language is Chinese", async () => {
    await i18n.changeLanguage("zh");
    renderWithI18n();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: i18n.t("language.changeLanguage") })
    );
    const englishItem = findDropdownItemByText("English");
    const chineseItem = findDropdownItemByText("中文");
    expect(englishItem).not.toBeNull();
    expect(chineseItem).not.toBeNull();
    expect(chineseItem!).toHaveClass("active");
    expect(englishItem!).not.toHaveClass("active");
  });
});
