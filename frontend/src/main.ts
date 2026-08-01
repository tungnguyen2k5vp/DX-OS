import { bootstrapApplication } from '@angular/platform-browser';
import { App } from './app/app';
import { createAppConfig } from './app/app.config';
import { AppConfig, isAppConfig } from './app/core/config/app-config';

async function loadRuntimeConfig(): Promise<AppConfig> {
  const response = await fetch('/config.json', { cache: 'no-store' });
  if (!response.ok) {
    throw new Error(`Không tải được runtime config (${response.status}).`);
  }

  const config: unknown = await response.json();
  if (!isAppConfig(config)) {
    throw new Error('Runtime config không hợp lệ.');
  }
  return config;
}

loadRuntimeConfig()
  .then((config) => bootstrapApplication(App, createAppConfig(config)))
  .catch((error: unknown) => {
    const message = error instanceof Error ? error.message : 'Không thể khởi động ứng dụng.';
    const root = document.querySelector('app-root');
    if (root) {
      root.innerHTML = `<main class="bootstrap-error"><h1>DX-OS chưa thể khởi động</h1><p>${escapeHtml(message)}</p></main>`;
    }
    console.error('Application bootstrap failed.');
  });

function escapeHtml(value: string): string {
  const element = document.createElement('span');
  element.textContent = value;
  return element.innerHTML;
}
