import { InjectionToken } from '@angular/core';

export interface AppConfig {
  apiBaseUrl: string;
  metabaseUrl: string;
  oidc: {
    url: string;
    realm: string;
    clientId: string;
  };
}

export const APP_CONFIG = new InjectionToken<AppConfig>('APP_CONFIG');

export function isAppConfig(value: unknown): value is AppConfig {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const candidate = value as Partial<AppConfig>;
  return (
    typeof candidate.apiBaseUrl === 'string' &&
    candidate.apiBaseUrl.startsWith('http') &&
    typeof candidate.metabaseUrl === 'string' &&
    candidate.metabaseUrl.startsWith('http') &&
    !!candidate.oidc &&
    typeof candidate.oidc.url === 'string' &&
    typeof candidate.oidc.realm === 'string' &&
    typeof candidate.oidc.clientId === 'string'
  );
}
