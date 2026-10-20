/// <reference types="node" />
import { defineRovoApp } from '@rovo/config/vite';

export default defineRovoApp({
  app: 'rider',
  root: import.meta.dirname,
  port: 5175,
  pwa: {
    name: 'rovo Delivery Partner',
    shortName: 'rovo Rider',
    description: 'Accept and deliver rovo orders',
    orientation: 'portrait',
    backgroundColor: '#FFFFFF',
  },
});
