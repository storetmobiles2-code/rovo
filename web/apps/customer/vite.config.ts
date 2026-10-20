/// <reference types="node" />
import { defineRovoApp } from '@rovo/config/vite';

export default defineRovoApp({
  app: 'customer',
  root: import.meta.dirname,
  port: 5173,
  pwa: {
    name: 'rovo — Food delivery',
    shortName: 'rovo',
    description: "Order food from Mahabubnagar's favourite restaurants",
    orientation: 'portrait',
    backgroundColor: '#FFFFFF',
  },
});
