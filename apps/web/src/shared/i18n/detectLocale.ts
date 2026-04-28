import type { Locale } from "i18n";

export function detectLocale(browserLanguage: string): Locale {
  if (browserLanguage.toLowerCase().startsWith("pt")) {
    return "pt-BR";
  }

  return "en";
}
