# ---- Build Stage ----
FROM node:24-alpine AS builder

WORKDIR /app

# Keep pnpm aligned with package.json and GitHub Actions.
RUN npm install -g pnpm@11.15.1

# Cache dependency installation separately from application sources.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
ENV CI=true \
    HUSKY=0
RUN pnpm install --frozen-lockfile

COPY . .

ARG VITE_BASE_URL=/
ENV VITE_BASE_URL=$VITE_BASE_URL

RUN pnpm build

# ---- Runtime Stage ----
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
