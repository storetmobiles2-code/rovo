/// <reference types="node" />
import { defineRovoApp } from '@rovo/config/vite';

// Admin has NO service worker (doc 18 §1): avoids caching sensitive data and keeps updates simple.
export default defineRovoApp({
  app: 'admin',
  root: import.meta.dirname,
  port: 5176,
  pwa: false,
});
