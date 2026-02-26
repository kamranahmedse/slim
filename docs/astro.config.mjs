import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  site: 'https://slim.sh',
  base: '/',
  output: 'static',
  outDir: './dist',
  vite: {
    plugins: [tailwindcss()],
  },
});
