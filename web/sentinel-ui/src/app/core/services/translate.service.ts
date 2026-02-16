import { Injectable, signal, Signal } from '@angular/core';
import trTranslations from '../../../i18n/tr.json';
import enTranslations from '../../../i18n/en.json';

interface Translations {
  [key: string]: any;
}

@Injectable({
  providedIn: 'root'
})
export class TranslateService {
  private readonly STORAGE_KEY = 'sentinel-language';
  private readonly DEFAULT_LANGUAGE = 'tr';

  private translations: Record<string, Translations> = {
    tr: trTranslations,
    en: enTranslations
  };

  private currentLanguage = signal<string>(this.DEFAULT_LANGUAGE);

  constructor() {
    this.loadLanguageFromStorage();
  }

  /**
   * Get translated string by key with optional parameter interpolation
   * @param key Dot-notation key (e.g., 'setup.watchers')
   * @param params Optional parameters for interpolation (e.g., {total: '150'})
   * @returns Translated string or the key itself if not found
   */
  get(key: string, params?: Record<string, string>): string {
    const lang = this.currentLanguage();
    const translation = this.getNestedValue(this.translations[lang], key);

    if (translation === undefined) {
      console.warn(`Translation key not found: ${key}`);
      return key;
    }

    if (typeof translation !== 'string') {
      console.warn(`Translation key is not a string: ${key}`);
      return key;
    }

    // Interpolate parameters if provided
    if (params) {
      return this.interpolate(translation, params);
    }

    return translation;
  }

  /**
   * Set the current language and persist to localStorage
   * @param lang Language code ('tr' or 'en')
   */
  setLanguage(lang: string): void {
    if (!this.translations[lang]) {
      console.warn(`Language not supported: ${lang}`);
      return;
    }

    this.currentLanguage.set(lang);
    localStorage.setItem(this.STORAGE_KEY, lang);
  }

  /**
   * Get the current language signal
   * @returns Signal containing current language code
   */
  getCurrentLanguage(): Signal<string> {
    return this.currentLanguage.asReadonly();
  }

  /**
   * Get nested value from object using dot notation
   * @param obj Object to search
   * @param path Dot-notation path (e.g., 'setup.watchers')
   * @returns Value at path or undefined
   */
  private getNestedValue(obj: any, path: string): any {
    return path.split('.').reduce((acc, part) => acc?.[part], obj);
  }

  /**
   * Interpolate parameters in translation string
   * Replaces {{param}} with values from params object
   * @param text Translation string with {{placeholders}}
   * @param params Object with replacement values
   * @returns Interpolated string
   */
  private interpolate(text: string, params: Record<string, string>): string {
    return text.replace(/\{\{(\w+)\}\}/g, (match, key) => {
      return params[key] !== undefined ? params[key] : match;
    });
  }

  /**
   * Load saved language from localStorage on init
   */
  private loadLanguageFromStorage(): void {
    const saved = localStorage.getItem(this.STORAGE_KEY);
    if (saved && this.translations[saved]) {
      this.currentLanguage.set(saved);
    }
  }
}
