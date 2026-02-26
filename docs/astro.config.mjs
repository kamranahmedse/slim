import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  site: 'https://kamranahmedse.github.io',
  base: '/slim',
  output: 'static',
  outDir: './dist',
  vite: {
    plugins: [tailwindcss()],
  },
});
