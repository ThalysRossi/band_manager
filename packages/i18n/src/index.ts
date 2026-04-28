export type Locale = 'en' | 'pt-BR'

export type TranslationKey =
  | 'app.title'
  | 'nav.inventory'
  | 'nav.merchBooth'
  | 'nav.reports'
  | 'nav.calendar'
  | 'status.backendReady'

export type TranslationDictionary = Record<TranslationKey, string>

export const translations: Record<Locale, TranslationDictionary> = {
  en: {
    'app.title': 'Band Manager',
    'nav.inventory': 'Inventory',
    'nav.merchBooth': 'Merch Booth',
    'nav.reports': 'Reports',
    'nav.calendar': 'Calendar',
    'status.backendReady': 'Backend foundation is ready'
  },
  'pt-BR': {
    'app.title': 'Band Manager',
    'nav.inventory': 'Estoque',
    'nav.merchBooth': 'Banca',
    'nav.reports': 'Relatorios',
    'nav.calendar': 'Calendario',
    'status.backendReady': 'Base do backend pronta'
  }
}
