FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.28.1 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY frontend/ ./
RUN pnpm run build && pnpm prune --prod --ignore-scripts

FROM node:22-alpine
WORKDIR /app
COPY --from=build --chown=node:node /app/package.json ./
COPY --from=build --chown=node:node /app/build ./build
COPY --from=build --chown=node:node /app/node_modules ./node_modules
USER node
ENV HOST=0.0.0.0 PORT=3000 NODE_ENV=production BODY_SIZE_LIMIT=12M
EXPOSE 3000
CMD ["node", "build"]
