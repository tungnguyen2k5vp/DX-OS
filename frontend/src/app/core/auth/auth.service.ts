import { computed, inject, Injectable, signal } from '@angular/core';
import Keycloak, { KeycloakTokenParsed } from 'keycloak-js';
import { APP_CONFIG } from '../config/app-config';

interface DxToken extends KeycloakTokenParsed {
  preferred_username?: string;
  email?: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly config = inject(APP_CONFIG);
  private readonly keycloak = new Keycloak(this.config.oidc);
  private readonly authenticatedState = signal(false);
  private readonly tokenClaims = signal<DxToken | null>(null);

  readonly authenticated = this.authenticatedState.asReadonly();
  readonly username = computed(
    () => this.tokenClaims()?.preferred_username ?? this.tokenClaims()?.sub ?? 'Người dùng',
  );
  readonly email = computed(() => this.tokenClaims()?.email ?? '');
  readonly roles = computed(() => this.tokenClaims()?.realm_access?.roles ?? []);

  async initialize(): Promise<void> {
    const authenticated = await this.keycloak.init({
      onLoad: 'login-required',
      pkceMethod: 'S256',
      checkLoginIframe: false,
    });

    this.authenticatedState.set(authenticated);
    this.tokenClaims.set((this.keycloak.tokenParsed as DxToken | undefined) ?? null);

    if (!authenticated) {
      await this.login();
    }
  }

  async login(): Promise<void> {
    await this.keycloak.login({
      redirectUri: window.location.origin + '/dashboard',
    });
  }

  async logout(): Promise<void> {
    this.authenticatedState.set(false);
    this.tokenClaims.set(null);
    await this.keycloak.logout({ redirectUri: window.location.origin });
  }

  async accessToken(): Promise<string | null> {
    if (!this.keycloak.authenticated) {
      return null;
    }

    await this.keycloak.updateToken(30);
    this.tokenClaims.set((this.keycloak.tokenParsed as DxToken | undefined) ?? null);
    return this.keycloak.token ?? null;
  }
}
