import { APP_BASE_HREF } from '@angular/common';
import { CommonEngine } from '@angular/ssr/node';
import express from 'express';
import { fileURLToPath } from 'node:url';
import { dirname, join, resolve } from 'node:path';
import bootstrap from './src/main.server';
import { createProxyMiddleware } from 'http-proxy-middleware';

// The Express app is exported so that it can be used by serverless Functions.
export function app(): express.Express {
  const server = express();
  const serverDistFolder = dirname(fileURLToPath(import.meta.url));
  const browserDistFolder = resolve(serverDistFolder, '../browser');
  const indexHtml = join(serverDistFolder, 'index.server.html');

  const allowedHosts = [
    'localhost',
    'localhost:4000',
    '127.0.0.1',
    'freq-show.adamlacasse.dev',
    'freq-show-frontend.onrender.com',
  ];
  if (process.env['RENDER_EXTERNAL_HOSTNAME']) {
    allowedHosts.push(process.env['RENDER_EXTERNAL_HOSTNAME']);
  }

  const commonEngine = new CommonEngine({
    allowedHosts,
  });

  server.set('trust proxy', 1);
  server.set('view engine', 'html');
  server.set('views', browserDistFolder);

  // ---- API proxy ----
  const apiTarget = process.env['API_BASE_URL'] || process.env['BACKEND_URL'] || 'http://localhost:8080';

  server.use(
    '/api',
    createProxyMiddleware({
      target: apiTarget,
      changeOrigin: true,
      pathRewrite: { '^/api': '' },
      on: {
        error: (err: any, req: any, res: any) => {
          console.error(`[API Proxy Error] Failed to proxy ${req.method} ${req.url} to ${apiTarget}:`, err);
          if (res && !res.headersSent) {
            res.status(502).json({ error: 'Backend proxy error', details: err.message });
          }
        }
      }
    })
  );

  // Serve static files from /browser.
  //
  // fallthrough:false is load-bearing. Without it, a request for a bundle that
  // isn't on disk falls through to the Angular catch-all below and gets back a
  // 200 with `text/html`. Browsers enforce strict MIME checking on module
  // scripts, so `<script type="module" src="main-<hash>.js">` is rejected, the
  // client never bootstraps, and the page still *looks* fine because SSR
  // already painted it — a silent, total loss of interactivity with nothing in
  // the server logs. Worse, that 200 is cacheable, so a CDN will happily pin
  // HTML under a .js URL for the full max-age below. Fail loudly instead.
  server.get(
    '*.*',
    express.static(browserDistFolder, {
      maxAge: '1y',
      fallthrough: false,
    })
  );

  // All regular routes use the Angular engine
  server.get('*', (req, res, next) => {
    const { protocol, originalUrl, baseUrl, headers } = req;

    commonEngine
      .render({
        bootstrap,
        documentFilePath: indexHtml,
        url: `${protocol}://${headers.host}${originalUrl}`,
        publicPath: browserDistFolder,
        providers: [{ provide: APP_BASE_HREF, useValue: baseUrl }],
      })
      .then((html) => {
        // Server-rendered HTML references hashed bundles that change on every
        // deploy, so it must never be cached at the edge. A stale cached
        // document points at bundles that no longer exist on the origin.
        res.set('Cache-Control', 'no-store');
        res.send(html);
      })
      .catch((err) => next(err));
  });

  // Keep asset 404s as plain 404s rather than letting Express's default handler
  // render an HTML error page (which can leak a stack trace when NODE_ENV isn't
  // set to production, as is easy to miss on Render).
  server.use(
    (
      err: any,
      _req: express.Request,
      res: express.Response,
      next: express.NextFunction
    ) => {
      if (res.headersSent) {
        return next(err);
      }
      const status = err?.status ?? err?.statusCode ?? 500;
      if (status === 404) {
        res.status(404).type('text/plain').send('Not Found');
        return;
      }
      console.error('[SSR Error]', err);
      res.status(500).type('text/plain').send('Internal Server Error');
    }
  );

  return server;
}

function run(): void {
  const port = process.env['PORT'] || 4000;

  // Start up the Node server
  const server = app();
  server.listen(port, () => {
    console.log(`Node Express server listening on http://localhost:${port}`);
  });
}

run();
