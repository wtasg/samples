import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1, // run sequentially to avoid state collisions in the V8 server
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:60001',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'go run src/server/main.go',
    url: 'http://localhost:60001',
    reuseExistingServer: true,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
