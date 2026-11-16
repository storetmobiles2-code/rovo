import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { registerSW } from 'virtual:pwa-register';
import { toast } from '@rovo/ui';
import { i18n } from '@rovo/i18n';
import { App } from './app';
import './styles.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Prompt-to-reload on a new service worker (doc 18 §5). Never auto-reloads the customer app.
const updateSW = registerSW({
  onNeedRefresh() {
    toast({
      title: i18n.t('pwa.updateAvailable'),
      tone: 'info',
      duration: Infinity,
      action: { label: i18n.t('actions.reload'), onClick: () => void updateSW(true) },
    });
  },
});
