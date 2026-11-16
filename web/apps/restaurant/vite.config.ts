/// <reference types="node" />
import { defineRovoApp } from '@rovo/config/vite';

export default defineRovoApp({
  app: 'restaurant',
  root: import.meta.dirname,
  port: 5174,
  pwa: {
    name: 'rovo Partner',
    shortName: 'rovo Partner',
    description: 'Manage orders and menu for your restaurant',
    orientation: 'any',
    backgroundColor: '#0C1825',
  },
});
