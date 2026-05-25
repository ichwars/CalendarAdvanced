import { createServer } from 'vite';

const apiTarget = process.env.VITE_API_TARGET ?? 'http://localhost:8090';

const server = await createServer({
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': apiTarget,
      '/dav': apiTarget
    }
  }
});

await server.listen();
server.printUrls();

process.on('SIGINT', async () => {
  await server.close();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  await server.close();
  process.exit(0);
});

setInterval(() => {}, 1 << 30);
