# AmiyaEden React frontend

`static-react/` is the React 19 + TypeScript + Vite frontend. It is built, verified, and deployed as the independent `frontend-react` image alongside the Vue `frontend` image in `static/`.

## Toolchain

- Node.js 24
- pnpm 11.15.1
- React 19 and React Router 7
- Vite 8 and TypeScript 6
- Tailwind CSS 4 and shadcn/ui
- Vitest, Testing Library, and ESLint

## Local development

```bash
pnpm install
pnpm dev
```

The development server reads `.env.development`, listens on port `5173`, and proxies `/api/*` to `VITE_API_PROXY_URL` (`http://localhost:8080` by default).

## Verification

```bash
pnpm lint
pnpm exec tsc -b
pnpm test
pnpm check:api-contract
pnpm build
```

These commands are enforced for `static-react/**` changes by `.github/workflows/verify-ci.yaml`.

## Container deployment

The multi-stage `Dockerfile` builds the Vite application with Node and serves `dist/` from Nginx. Nginx proxies `/api/*` to the Compose service `backend:8080`. From the repository root, run:

```bash
docker compose -f docker-compose.example.yml up -d backend frontend frontend-react
```

- Vue remains available on host port `80`.
- React is available on host port `3000` through the `frontend-react` service.
- The React image is `xiaomaitx/amiya-eden-frontend-react`.
- `main` publishes `latest` and short-SHA tags; `preview` publishes the `preview` tag.

`VITE_BASE_URL` can be supplied as a Docker build argument when deploying the compiled assets below a non-root base path; the default is `/`.
